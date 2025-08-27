package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ provider.Provider = &KubeVirtProvider{}
)

// KubeVirtProvider is the provider implementation.
type KubeVirtProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// KubeVirtProviderModel describes the provider data model.
type KubeVirtProviderModel struct {
	Namespace types.String `tfsdk:"namespace"`
}

func (p *KubeVirtProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "kubevirt"
	resp.Version = p.version
}

func (p *KubeVirtProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with KubeVirt to manage Virtual Machines.",
		Attributes: map[string]schema.Attribute{
			"namespace": schema.StringAttribute{
				Description: "The default namespace for KubeVirt resources. This can also be specified per-resource.",
				Optional:    true,
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
	// if data.Namespace.IsNull() { /* ... */ }

	// Example client configuration for data sources and resources
	// client := &http.Client{}
	// resp.DataSourceData = client
	// resp.ResourceData = client
}

func (p *KubeVirtProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewKubeVirtVMResource,
		NewKubernetesPodResource,
	}
}

func (p *KubeVirtProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewExampleDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &KubeVirtProvider{
			version: version,
		}
	}
}
