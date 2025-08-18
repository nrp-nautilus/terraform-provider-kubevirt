package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &KubernetesPodResource{}
var _ resource.ResourceWithImportState = &KubernetesPodResource{}

func NewKubernetesPodResource() resource.Resource {
	return &KubernetesPodResource{}
}

// KubernetesPodResource defines the resource implementation.
type KubernetesPodResource struct {
	client *kubernetes.Clientset
}

// KubernetesPodResourceModel describes the resource data model.
type KubernetesPodResourceModel struct {
	Id                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Namespace         types.String `tfsdk:"namespace"`
	Image             types.String `tfsdk:"image"`
	Command           types.List   `tfsdk:"command"`
	Args              types.List   `tfsdk:"args"`
	ContainerName     types.String `tfsdk:"container_name"`
	PodStatus         types.String `tfsdk:"pod_status"`
	CreationTimestamp types.String `tfsdk:"creation_timestamp"`
}

func (r *KubernetesPodResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_pod"
}

func (r *KubernetesPodResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Kubernetes pod resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Pod identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the pod",
				Required:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace",
				Required:            true,
			},
			"image": schema.StringAttribute{
				MarkdownDescription: "Container image to use",
				Required:            true,
			},
			"command": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Command to run in the container",
				Optional:            true,
			},
			"args": schema.ListAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Arguments for the command",
				Optional:            true,
			},
			"container_name": schema.StringAttribute{
				MarkdownDescription: "Name of the container",
				Optional:            true,
			},
			"pod_status": schema.StringAttribute{
				MarkdownDescription: "Current status of the pod",
				Computed:            true,
			},
			"creation_timestamp": schema.StringAttribute{
				MarkdownDescription: "When the pod was created",
				Computed:            true,
			},
		},
	}
}

func (r *KubernetesPodResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	// The provider data is the provider instance itself
	_, ok := req.ProviderData.(*KubeVirtProvider)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *KubeVirtProvider, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Get Kubernetes client from provider
	config, err := rest.InClusterConfig()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get Kubernetes config",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create Kubernetes client",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	r.client = clientset

	// Log that we're configured
	tflog.Info(ctx, "Configured Kubernetes pod resource")
}

func (r *KubernetesPodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubernetesPodResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Convert Terraform types to Go types
	var command []string
	if !data.Command.IsNull() && !data.Command.IsUnknown() {
		resp.Diagnostics.Append(data.Command.ElementsAs(ctx, &command, false)...)
	}

	var args []string
	if !data.Args.IsNull() && !data.Args.IsUnknown() {
		resp.Diagnostics.Append(data.Args.ElementsAs(ctx, &args, false)...)
	}

	containerName := data.ContainerName.ValueString()
	if containerName == "" {
		containerName = "main"
	}

	// Create the pod
	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      data.Name.ValueString(),
			Namespace: data.Namespace.ValueString(),
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:  containerName,
					Image: data.Image.ValueString(),
				},
			},
		},
	}

	// Add command and args if specified
	if len(command) > 0 {
		pod.Spec.Containers[0].Command = command
	}
	if len(args) > 0 {
		pod.Spec.Containers[0].Args = args
	}

	// Create the pod in Kubernetes
	createdPod, err := r.client.CoreV1().Pods(data.Namespace.ValueString()).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create pod",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	// Set the computed values
	data.Id = types.StringValue(string(createdPod.UID))
	data.PodStatus = types.StringValue(string(createdPod.Status.Phase))
	data.CreationTimestamp = types.StringValue(createdPod.CreationTimestamp.String())

	// Write logs using the tflog package
	tflog.Trace(ctx, "created a kubernetes pod resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesPodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubernetesPodResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Get the pod from Kubernetes
	pod, err := r.client.CoreV1().Pods(data.Namespace.ValueString()).Get(ctx, data.Name.ValueString(), metav1.GetOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to read pod",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	// Update the model with current values
	data.PodStatus = types.StringValue(string(pod.Status.Phase))
	data.CreationTimestamp = types.StringValue(pod.CreationTimestamp.String())

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesPodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KubernetesPodResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For simplicity, we'll delete and recreate the pod
	// In a production provider, you'd want to implement proper updates
	resp.Diagnostics.AddWarning(
		"Pod updates not implemented",
		"Pod updates require deletion and recreation",
	)
}

func (r *KubernetesPodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KubernetesPodResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Delete the pod from Kubernetes
	err := r.client.CoreV1().Pods(data.Namespace.ValueString()).Delete(ctx, data.Name.ValueString(), metav1.DeleteOptions{})
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to delete pod",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}
}

func (r *KubernetesPodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
