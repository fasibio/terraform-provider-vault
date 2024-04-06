// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"net/http"

	client "github.com/cryptvault-cloud/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &VaultCloud{}

// VaultCloud defines the provider implementation.
type VaultCloud struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// VaultCloudProviderModel describes the provider data model.
type VaultCloudProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
}

func (p *VaultCloud) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "cryptvault_cloud"
	resp.Version = p.version
}

func (p *VaultCloud) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "vault endpoint",
				Optional:            true,
			},
		},
	}
}

func (p *VaultCloud) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data VaultCloudProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Configuration values are now available.
	if data.Endpoint.IsNull() {
		data.Endpoint = basetypes.NewStringValue("https://api.cryptvault.cloud/query")
	}

	// Example client configuration for data sources and resources
	client := client.NewApi(data.Endpoint.ValueString(), http.DefaultClient)
	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *VaultCloud) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewVaultResource,
		NewIdentityResource,
		NewValueResource,
		NewKeyPairResource,
	}
}

func (p *VaultCloud) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewVaultDataSource,
		NewIdentityDataSource,
		NewValueDataSource,
		NewPublicKeyDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &VaultCloud{
			version: version,
		}
	}
}
