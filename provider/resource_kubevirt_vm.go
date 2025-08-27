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
	_ resource.Resource                = &KubeVirtVMResource{}
	_ resource.ResourceWithConfigure   = &KubeVirtVMResource{}
	_ resource.ResourceWithImportState = &KubeVirtVMResource{}
)

// KubeVirtVMResource is the resource implementation.
type KubeVirtVMResource struct {
	// Add any fields you need here
}

// KubeVirtVMResourceModel describes the resource data model.
type KubeVirtVMResourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Namespace types.String `tfsdk:"namespace"`
	Image     types.String `tfsdk:"image"`
	Memory    types.String `tfsdk:"memory"`
	CPU       types.Int64  `tfsdk:"cpu"`
}

func (r *KubeVirtVMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kubevirt_vm"
}

func (r *KubeVirtVMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Version: 1,
		Description: "Manages a KubeVirt Virtual Machine",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The ID of the VM",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "The name of the VM",
				Required:    true,
			},
			"namespace": schema.StringAttribute{
				Description: "The namespace of the VM",
				Required:    true,
			},
			"image": schema.StringAttribute{
				Description: "The container image for the VM",
				Required:    true,
			},
			"memory": schema.StringAttribute{
				Description: "The memory allocation for the VM",
				Required:    true,
			},
			"cpu": schema.Int64Attribute{
				Description: "The CPU allocation for the VM",
				Required:    true,
			},
		},
	}
}

func (r *KubeVirtVMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// No configuration needed for now
}

func (r *KubeVirtVMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data KubeVirtVMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For now, just set the ID to the name
	data.ID = data.Name

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubeVirtVMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data KubeVirtVMResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// For now, just keep the existing state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubeVirtVMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data KubeVirtVMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *KubeVirtVMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// No cleanup needed for now
}

func (r *KubeVirtVMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func NewKubeVirtVMResource() resource.Resource {
	return &KubeVirtVMResource{}
}
