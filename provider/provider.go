package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure KubeVirtProvider satisfies various provider interfaces.
var _ provider.Provider = &KubeVirtProvider{}

// KubeVirtProvider defines the provider implementation.
type KubeVirtProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// KubeVirtProviderModel describes the provider data model.
type KubeVirtProviderModel struct {
	Endpoint  types.String `tfsdk:"endpoint"`
	Namespace types.String `tfsdk:"namespace"`
}

func (p *KubeVirtProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubevirt"
	resp.Version = p.version
}

func (p *KubeVirtProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Example provider attribute",
				Optional:            true,
			},
			"namespace": schema.StringAttribute{
				MarkdownDescription: "Kubernetes namespace to operate in",
				Optional:            true,
				Default:             stringdefault.StaticString("terraform-dev"),
			},
		},
	}
}

func (p *KubeVirtProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data KubeVirtProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	// data.Endpoint.ValueString()

	// Example client configuration for data sources and resources
	resp.DataSourceData = data
	resp.ResourceData = data
}

func (p *KubeVirtProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewHelloWorldResource,
	}
}

func (p *KubeVirtProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewHelloWorldDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KubeVirtProvider{
			version: version,
		}
	}
}
