package provider

import (
	"context"
	"fmt"

	client "github.com/cryptvault-cloud/api"
	"github.com/cryptvault-cloud/helper"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSourceWithConfigure = &IdentityDataSource{}

func NewIdentityDataSource() datasource.DataSource {
	return &IdentityDataSource{}
}

type IdentityDataSource struct {
	client client.ApiHandler
}

type IdentityDataSourceModel struct {
	Id         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	PublicKey  types.String `tfsdk:"public_key"`
	PrivateKey types.String `tfsdk:"private_key"`
	VaultID    types.String `tfsdk:"vault_id"`
}

func (d *IdentityDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (d *IdentityDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Load and read an already exist identity over vaultId and identityPrivateKey ",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of identity",
				Description:         "id of identity",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of identity",
				Description:         "Name of identity",
				Computed:            true,
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public Key of identity",
				Description:         "Public Key of identity",
				Computed:            true,
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Private Key of identity",
				Description:         "Private Key of identity",
				Required:            true,
				Sensitive:           true,
			},
			"vault_id": schema.StringAttribute{
				MarkdownDescription: "ID of used vault",
				Description:         "ID of used vault",
				Required:            true,
			},
		},
	}
}

func (d *IdentityDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, err := getClient(&req)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.: %v", req.ProviderData, err),
		)

		return
	}

	d.client = client
}

func (d *IdentityDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IdentityDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	private_key, err := helper.GetPrivateKeyFromB64String(data.PrivateKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Private key is not an ecdsa.Private key: %s", err.Error()), "")
		return
	}

	vault_id := data.VaultID.ValueString()

	pApi := d.client.GetProtectedApi(private_key, vault_id)
	pubToken, err := helper.NewBase64PublicPem(&private_key.PublicKey)
	if err != nil {
		resp.Diagnostics.AddError("Public key can not be pemed", err.Error())
		return
	}

	token_id, err := pubToken.GetIdentityId(vault_id)
	if err != nil {
		resp.Diagnostics.AddError("Identity id can not be generated", err.Error())
	}
	identity, err := pApi.GetIdentity(token_id)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Identity can not be fetched from API Id: %s", token_id), err.Error())
		return
	}

	data.Id = types.StringValue(identity.Id)
	data.PublicKey = types.StringValue(string(identity.PublicKey))
	data.Name = types.StringValue(*identity.Name)

	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
