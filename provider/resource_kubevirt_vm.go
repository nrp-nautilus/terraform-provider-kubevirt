package provider

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8sschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ resource.Resource                = &KubeVirtVMResource{}
	_ resource.ResourceWithConfigure   = &KubeVirtVMResource{}
	_ resource.ResourceWithImportState = &KubeVirtVMResource{}
)

// KubeVirtVMResource is the resource implementation.
type KubeVirtVMResource struct {
	client        *kubernetes.Clientset
	dynamicClient dynamic.Interface
	namespace     string
}

// KubeVirtVMResourceModel describes the resource data model.
type KubeVirtVMResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Namespace         types.String `tfsdk:"namespace"`
	Image             types.String `tfsdk:"image"`
	Memory            types.String `tfsdk:"memory"`
	CPU               types.Int64  `tfsdk:"cpu"`
	MachineType       types.String `tfsdk:"machine_type"`
	Architecture      types.String `tfsdk:"architecture"`
	Hugepages         types.String `tfsdk:"hugepages"`
	SidecarHook       types.String `tfsdk:"sidecar_hook"`
	NodeSelector      types.Map    `tfsdk:"node_selector"`
	Tolerations       types.List   `tfsdk:"tolerations"`
	HostDevices       types.List   `tfsdk:"host_devices"`
	USBDevices        types.List   `tfsdk:"usb_devices"`
	NetworkInterfaces types.List   `tfsdk:"network_interfaces"`
	CloudInit         types.String `tfsdk:"cloud_init"`
	CoderAgentToken   types.String `tfsdk:"coder_agent_token"`
	VMStatus          types.String `tfsdk:"vm_status"`
	CreationTimestamp types.String `tfsdk:"creation_timestamp"`
	WorkspaceTransition types.String `tfsdk:"workspace_transition"`
}

func NewKubeVirtVMResource() resource.Resource {
	return &KubeVirtVMResource{}
}

// Metadata returns the resource type name.
func (r *KubeVirtVMResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubevirt_vm"
}

// Schema defines the schema for the resource.
func (r *KubeVirtVMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a KubeVirt VirtualMachine",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the VM",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Name of the VirtualMachine",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"namespace": schema.StringAttribute{
				Description: "Kubernetes namespace for the VM",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"image": schema.StringAttribute{
				Description: "Container image for the VM",
				Required:    true,
			},
			"memory": schema.StringAttribute{
				Description: "Memory allocation (e.g., '2Gi')",
				Required:    true,
			},
			"cpu": schema.Int64Attribute{
				Description: "Number of CPU cores",
				Required:    true,
			},
			"machine_type": schema.StringAttribute{
				Description: "QEMU machine type (e.g., 'q35')",
				Optional:    true,
			},
			"architecture": schema.StringAttribute{
				Description: "CPU architecture (e.g., 'x86_64')",
				Optional:    true,
			},
			"hugepages": schema.StringAttribute{
				Description: "Hugepages configuration (e.g., '1Gi')",
				Optional:    true,
			},
			"sidecar_hook": schema.StringAttribute{
				Description: "Sidecar hook script name (ConfigMap)",
				Optional:    true,
			},
			"node_selector": schema.MapAttribute{
				Description: "Node selector labels",
				ElementType: types.StringType,
				Optional:    true,
			},
			"tolerations": schema.ListAttribute{
				Description: "Tolerations for node scheduling",
				ElementType: types.StringType,
				Optional:    true,
			},
			"host_devices": schema.ListAttribute{
				Description: "PCI host devices to attach",
				ElementType: types.StringType,
				Optional:    true,
			},
			"usb_devices": schema.ListAttribute{
				Description: "USB devices to attach",
				ElementType: types.StringType,
				Optional:    true,
			},
			"network_interfaces": schema.ListAttribute{
				Description: "Network interface configurations",
				ElementType: types.StringType,
				Optional:    true,
			},
			"cloud_init": schema.StringAttribute{
				Description: "Cloud-init user data",
				Optional:    true,
			},
			"coder_agent_token": schema.StringAttribute{
				Description: "Token for the Coder agent to authenticate with the VM",
				Optional:    true,
			},
			"vm_status": schema.StringAttribute{
				Description: "Current status of the VM",
				Computed:    true,
			},
			"creation_timestamp": schema.StringAttribute{
				Description: "Timestamp when the VM was created",
				Computed:    true,
			},
			"workspace_transition": schema.StringAttribute{
				Description: "Workspace transition state for lifecycle management (start, stop, delete)",
				Optional:    true,
			},
		},
	}
}

// Configure adds the provider configured client to the resource.
func (r *KubeVirtVMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	provider, ok := req.ProviderData.(KubeVirtProviderModel)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected KubeVirtProviderModel, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	// Get the namespace from provider config
	namespace := "default"
	if !provider.Namespace.IsNull() && !provider.Namespace.IsUnknown() {
		namespace = provider.Namespace.ValueString()
	}

	// Create Kubernetes client
	config, err := rest.InClusterConfig()
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to get cluster config",
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

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to create dynamic client",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	r.client = clientset
	r.dynamicClient = dynamicClient
	r.namespace = namespace
}

// createVM is a helper method to create a VM without triggering the full Create flow
func (r *KubeVirtVMResource) createVM(ctx context.Context, data KubeVirtVMResourceModel, namespace string) (*unstructured.Unstructured, error) {
	// Create the VM
	vm := &unstructured.Unstructured{}
	vm.SetAPIVersion("kubevirt.io/v1")
	vm.SetKind("VirtualMachine")
	vm.SetName(data.Name.ValueString())
	vm.SetNamespace(namespace)

	// Create the VirtualMachine manifest
	vm.Object = map[string]interface{}{
		"apiVersion": "kubevirt.io/v1",
		"kind":       "VirtualMachine",
		"metadata": map[string]interface{}{
			"name":      data.Name.ValueString(),
			"namespace": namespace,
		},
		"spec": map[string]interface{}{
			"running": true,
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"kubevirt.io/vm": data.Name.ValueString(),
					},
				},
				"spec": map[string]interface{}{
					"domain": map[string]interface{}{
						"devices": map[string]interface{}{
							"disks": []map[string]interface{}{
								{
									"name": "containerdisk",
									"disk": map[string]interface{}{
										"bus": "virtio",
									},
								},
							},
							"interfaces": []map[string]interface{}{
								{
									"name": "default",
									"bridge": map[string]interface{}{},
								},
							},
						},
						"resources": map[string]interface{}{
							"requests": map[string]interface{}{
								"memory": data.Memory.ValueString(),
								"cpu":    data.CPU.ValueInt64(),
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "containerdisk",
							"containerDisk": map[string]interface{}{
								"image": data.Image.ValueString(),
							},
						},
					},
					"networks": []map[string]interface{}{
						{
							"name": "default",
							"pod":  map[string]interface{}{},
						},
					},
				},
			},
		},
	}

	// Add machine type if specified
	if !data.MachineType.IsNull() && !data.MachineType.IsUnknown() {
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["machine"] = map[string]interface{}{
			"type": data.MachineType.ValueString(),
		}
	}

	// Add architecture if specified
	if !data.Architecture.IsNull() && !data.Architecture.IsUnknown() {
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["cpu"] = map[string]interface{}{
			"architecture": data.Architecture.ValueString(),
		}
	}

	// Add hugepages if specified
	if !data.Hugepages.IsNull() && !data.Hugepages.IsUnknown() {
		hugepagesValue := data.Hugepages.ValueString()
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["resources"].(map[string]interface{})["requests"].(map[string]interface{})["hugepages-"+hugepagesValue] = hugepagesValue
		// Also set limits for hugepages (required by Kubernetes)
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["resources"].(map[string]interface{})["limits"] = map[string]interface{}{
			"hugepages-"+hugepagesValue: hugepagesValue,
		}
	}

	// Add sidecar hook if specified
	if !data.SidecarHook.IsNull() && !data.SidecarHook.IsUnknown() {
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["metadata"].(map[string]interface{})["annotations"] = map[string]interface{}{
			"hooks.kubevirt.io/hookSidecars": fmt.Sprintf(`[{"args":["--version","v1alpha2"],"configMap":{"hookPath":"/usr/bin/onDefineDomain","key":"%s.py","name":"%s"}}]`, data.SidecarHook.ValueString(), data.SidecarHook.ValueString()),
		}
	}

	// Add node selector if specified
	if !data.NodeSelector.IsNull() && !data.NodeSelector.IsUnknown() {
		var nodeSelector map[string]string
		data.NodeSelector.ElementsAs(ctx, &nodeSelector, false)
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["nodeSelector"] = nodeSelector
	}

	// Add tolerations if specified
	if !data.Tolerations.IsNull() && !data.Tolerations.IsUnknown() {
		var tolerations []string
		data.Tolerations.ElementsAs(ctx, &tolerations, false)
		tolObjects := make([]map[string]interface{}, len(tolerations))
		for i, tol := range tolerations {
			if strings.Contains(tol, ":") {
				// Parse key:effect format
				parts := strings.SplitN(tol, ":", 2)
				keyValue := parts[0]
				effect := parts[1]
				
				tolObj := map[string]interface{}{
					"effect": effect,
				}
				
				// Check if key has a value (key=value format)
				if strings.Contains(keyValue, "=") {
					kvParts := strings.SplitN(keyValue, "=", 2)
					tolObj["key"] = kvParts[0]
					tolObj["value"] = kvParts[1]
				} else {
					tolObj["key"] = keyValue
					// Use operator: Exists for key-only tolerations (like the working VM)
					tolObj["operator"] = "Exists"
				}
				
				tolObjects[i] = tolObj
			} else {
				// Fallback: treat as key only with Exists operator
				tolObjects[i] = map[string]interface{}{
					"key":      tol,
					"operator": "Exists",
					"effect":   "NoSchedule",
				}
			}
		}
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["tolerations"] = tolObjects
	}

	// Add host devices if specified
	if !data.HostDevices.IsNull() && !data.HostDevices.IsUnknown() {
		var devices []string
		data.HostDevices.ElementsAs(ctx, &devices, false)
		hostDevices := make([]map[string]interface{}, len(devices))
		for i, device := range devices {
			hostDevices[i] = map[string]interface{}{
				"name":       fmt.Sprintf("hostdevice-%d", i),
				"deviceName": device,
			}
		}
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["hostDevices"] = hostDevices
	}

	// Add USB devices if specified
	if !data.USBDevices.IsNull() && !data.USBDevices.IsUnknown() {
		var usbDevices []string
		data.USBDevices.ElementsAs(ctx, &usbDevices, false)
		usbObjects := make([]map[string]interface{}, len(usbDevices))
		for i, device := range usbDevices {
			usbObjects[i] = map[string]interface{}{
				"name":       fmt.Sprintf("usb-%d", i),
				"vendor":     "0x1234", // Default vendor ID
				"product":    "0x5678", // Default product ID
				"deviceName": device,
			}
		}
		vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["domain"].(map[string]interface{})["devices"].(map[string]interface{})["usb"] = usbObjects
	}

	// Add cloud-init if specified
	if !data.CloudInit.IsNull() && !data.CloudInit.IsUnknown() {
		// Build enhanced cloud-init that matches working ubuntu-vm pattern
		cloudInitData := data.CloudInit.ValueString()
		
		// If Coder agent token is provided, enhance the cloud-init with proper agent setup
		if !data.CoderAgentToken.IsNull() && !data.CoderAgentToken.IsUnknown() {
			// Create enhanced cloud-init with Coder agent (matching working ubuntu-vm pattern)
			enhancedCloudInit := fmt.Sprintf(`#cloud-config
%s

# Coder Agent Setup (following working ubuntu-vm pattern)
write_files:
  - path: /opt/coder/init
    permissions: "0755"
    content: |
      #!/bin/bash
      set -e
      
      # install and start code-server
      curl -fsSL https://code-server.dev/install.sh | sh -s -- --method=standalone --prefix=/tmp/code-server --version 4.11.0
      /tmp/code-server/bin/code-server --auth none --port 13337 >/tmp/code-server.log 2>&1 &
      
      # Start Coder agent
      exec coder agent --url https://coder-dev.nrp-nautilus.io --token %s
      
  - path: /etc/systemd/system/coder-agent.service
    permissions: "0644"
    content: |
      [Unit]
      Description=Coder Agent
      After=network-online.target
      Wants=network-online.target
      
      [Service]
      User=coder
      ExecStart=/opt/coder/init
      EnvironmentFile=/var/run/secrets/.coder-agent-token
      Restart=always
      RestartSec=10
      TimeoutStopSec=90
      KillMode=process
      OOMScoreAdjust=-900
      SyslogIdentifier=coder-agent
      
      [Install]
      WantedBy=multi-user.target

bootcmd:
  - mkdir -p /var/run/secrets
  - echo CODER_AGENT_TOKEN=%s > /var/run/secrets/.coder-agent-token

runcmd:
  - systemctl enable coder-agent
  - systemctl start coder-agent
  - echo "Coder agent setup complete!"`, 
				cloudInitData,
				data.CoderAgentToken.ValueString(),
				data.CoderAgentToken.ValueString())
			
			cloudInitData = enhancedCloudInit
		}
		
		// Check if cloud-init data exceeds the 2048 byte limit
		if len(cloudInitData) > 2048 {
			// Create a secret for the cloud-init data
			secretName := fmt.Sprintf("coder-%s-cloudinit", data.Name.ValueString())
			
			// Create the secret first
			secret := &unstructured.Unstructured{}
			secret.SetAPIVersion("v1")
			secret.SetKind("Secret")
			secret.SetName(secretName)
			secret.SetNamespace(namespace)
			secret.Object["data"] = map[string]interface{}{
				"userdata": base64.StdEncoding.EncodeToString([]byte(cloudInitData)),
			}
			
			// Create the secret
			secretGVR := k8sschema.GroupVersionResource{
				Group:    "",
				Version:  "v1",
				Resource: "secrets",
			}
			
			_, err := r.dynamicClient.Resource(secretGVR).Namespace(namespace).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil && !k8serrors.IsAlreadyExists(err) {
				return nil, fmt.Errorf("failed to create cloud-init secret: %w", err)
			}
			
			// Add the secret reference to the VM
			vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"] = append(
				vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"].([]map[string]interface{}),
				map[string]interface{}{
					"name": "cloudinitdisk",
					"cloudInitNoCloud": map[string]interface{}{
						"secretRef": map[string]interface{}{
							"name": secretName,
						},
					},
				},
			)
		} else {
			// Use inline cloud-init data for small configurations
			vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"] = append(
				vm.Object["spec"].(map[string]interface{})["template"].(map[string]interface{})["spec"].(map[string]interface{})["volumes"].([]map[string]interface{}),
				map[string]interface{}{
					"name": "cloudinitdisk",
					"cloudInitNoCloud": map[string]interface{}{
						"userData": cloudInitData,
					},
				},
			)
		}
	}

	// Create the VM in Kubernetes
	vmGVR := k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}

	createdVM, err := r.dynamicClient.Resource(vmGVR).Namespace(namespace).Create(ctx, vm, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create VirtualMachine: %w", err)
	}

	return createdVM, nil
}

// Create creates the resource and sets the initial Terraform state.
func (r *KubeVirtVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubeVirtVMResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use namespace from resource if specified, otherwise use provider default
	namespace := r.namespace
	if !data.Namespace.IsNull() && !data.Namespace.IsUnknown() {
		namespace = data.Namespace.ValueString()
	}

	// Check if this is a workspace start transition
	// Only create VM when workspace is starting
	if !data.WorkspaceTransition.IsNull() && !data.WorkspaceTransition.IsUnknown() {
		transition := data.WorkspaceTransition.ValueString()
		if transition != "start" {
			// Not starting, just return success without creating VM
			resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
			return
		}
	}

	// Create the VM
	vm, err := r.createVM(ctx, data, namespace)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create VirtualMachine", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(fmt.Sprintf("%s/%s", namespace, data.Name.ValueString()))
	data.VMStatus = types.StringValue("Created")
	data.CreationTimestamp = types.StringValue(vm.GetCreationTimestamp().Format(time.RFC3339))
	data.WorkspaceTransition = types.StringValue("start") // Indicate successful creation

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *KubeVirtVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubeVirtVMResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = r.namespace
	}

	// Get the VM from Kubernetes
	gvr := k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}

	vm, err := r.dynamicClient.Resource(gvr).Namespace(namespace).Get(ctx, data.Name.ValueString(), metav1.GetOptions{})
	if err != nil {
		if k8serrors.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Failed to read VirtualMachine",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	// Update the model with current values
	data.VMStatus = types.StringValue("Running") // This would need more sophisticated status checking
	data.CreationTimestamp = types.StringValue(vm.GetCreationTimestamp().Format(time.RFC3339))
	data.WorkspaceTransition = types.StringValue("start") // Assume running state means start transition

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *KubeVirtVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KubeVirtVMResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use namespace from resource if specified, otherwise use provider default
	namespace := r.namespace
	if !data.Namespace.IsNull() && !data.Namespace.IsUnknown() {
		namespace = data.Namespace.ValueString()
	}

	// Get the current state to check for transitions
	var state KubeVirtVMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if this is a workspace transition
	if !data.WorkspaceTransition.IsNull() && !data.WorkspaceTransition.IsUnknown() {
		oldTransition := state.WorkspaceTransition.ValueString()
		newTransition := data.WorkspaceTransition.ValueString()
		
		// Handle start transition
		if newTransition == "start" && oldTransition != "start" {
			// Starting the workspace - ensure VM is running
			vm, err := r.dynamicClient.Resource(k8sschema.GroupVersionResource{
				Group:    "kubevirt.io",
				Version:  "v1",
				Resource: "virtualmachines",
			}).Namespace(namespace).Get(ctx, data.Name.ValueString(), metav1.GetOptions{})
			
			if err != nil {
				if k8serrors.IsNotFound(err) {
					// VM doesn't exist, create it
					createdVM, err := r.createVM(ctx, data, namespace)
					if err != nil {
						resp.Diagnostics.AddError("Failed to create VM", err.Error())
						return
					}
					data.VMStatus = types.StringValue("Created")
					data.CreationTimestamp = types.StringValue(createdVM.GetCreationTimestamp().Format(time.RFC3339))
					resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
					return
				}
				resp.Diagnostics.AddError("Failed to get VM", err.Error())
				return
			}
			
			// VM exists, ensure it's running
			vm.Object["spec"].(map[string]interface{})["running"] = true
			_, err = r.dynamicClient.Resource(k8sschema.GroupVersionResource{
				Group:    "kubevirt.io",
				Version:  "v1",
				Resource: "virtualmachines",
			}).Namespace(namespace).Update(ctx, vm, metav1.UpdateOptions{})
			
			if err != nil {
				resp.Diagnostics.AddError("Failed to start VM", err.Error())
				return
			}
			
			data.VMStatus = types.StringValue("Running")
		}
		
		// Handle stop transition
		if newTransition == "stop" && oldTransition != "stop" {
			// Stopping the workspace - stop the VM
			vm, err := r.dynamicClient.Resource(k8sschema.GroupVersionResource{
				Group:    "kubevirt.io",
				Version:  "v1",
				Resource: "virtualmachines",
			}).Namespace(namespace).Get(ctx, data.Name.ValueString(), metav1.GetOptions{})
			
			if err != nil {
				if k8serrors.IsNotFound(err) {
					// VM doesn't exist, that's fine
					data.VMStatus = types.StringValue("Stopped")
					resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
					return
				}
				resp.Diagnostics.AddError("Failed to get VM", err.Error())
				return
			}
			
			// Stop the VM
			vm.Object["spec"].(map[string]interface{})["running"] = false
			_, err = r.dynamicClient.Resource(k8sschema.GroupVersionResource{
				Group:    "kubevirt.io",
				Version:  "v1",
				Resource: "virtualmachines",
			}).Namespace(namespace).Update(ctx, vm, metav1.UpdateOptions{})
			
			if err != nil {
				resp.Diagnostics.AddError("Failed to stop VM", err.Error())
				return
			}
			
			data.VMStatus = types.StringValue("Stopped")
		}
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *KubeVirtVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data KubeVirtVMResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	namespace := data.Namespace.ValueString()
	if namespace == "" {
		namespace = r.namespace
	}

	// Check if this is a workspace stop or delete transition
	// Only delete VM when workspace is stopping or deleting
	if !data.WorkspaceTransition.IsNull() && !data.WorkspaceTransition.IsUnknown() {
		transition := data.WorkspaceTransition.ValueString()
		if transition != "stop" && transition != "delete" {
			// Not stopping or deleting, just return success
			resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
			return
		}
	}

	// Delete the VM from Kubernetes
	gvr := k8sschema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}

	err := r.dynamicClient.Resource(gvr).Namespace(namespace).Delete(ctx, data.Name.ValueString(), metav1.DeleteOptions{})
	if err != nil && !k8serrors.IsNotFound(err) {
		resp.Diagnostics.AddError(
			"Failed to delete VirtualMachine",
			fmt.Sprintf("Error: %v", err),
		)
		return
	}

	tflog.Info(ctx, "VirtualMachine deleted successfully")
}

// ImportState imports the resource into Terraform state.
func (r *KubeVirtVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
