package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &KubernetesPodResource{}
	_ resource.ResourceWithConfigure   = &KubernetesPodResource{}
	_ resource.ResourceWithImportState = &KubernetesPodResource{}
)

// KubernetesPodResource is the resource implementation.
type KubernetesPodResource struct {
	// Add any fields you need here
}

// KubernetesPodResourceModel describes the resource data model.
type KubernetesPodResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
	Image     types.String `tfsdk:"image"`
}

func (r *KubernetesPodResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubernetes_pod"
}

func (r *KubernetesPodResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Kubernetes Pod",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the Pod",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the Pod",
				Required:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "The namespace of the Pod",
				Required:    true,
			},
			"image": schema.StringAttribute{
				Description: "The container image for the Pod",
				Required:    true,
			},
		},
	}
}

func (r *KubernetesPodResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// No configuration needed for now
}

func (r *KubernetesPodResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubernetesPodResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For now, just set the ID to the name
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesPodResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubernetesPodResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For now, just keep the existing state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesPodResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KubernetesPodResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubernetesPodResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No cleanup needed for now
}

func (r *KubernetesPodResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewKubernetesPodResource() resource.Resource {
	return &KubernetesPodResource{}
}
