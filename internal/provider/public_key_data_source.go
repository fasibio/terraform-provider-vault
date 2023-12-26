package provider

import (
	"context"

	"github.com/cryptvault-cloud/helper"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &PublicKeyDataSource{}

type PublicKeyDataSource struct {
}

type PublicKeyDataSourceModel struct {
	PublicKey types.String `tfsdk:"public_key"`
}

func NewPublicKeyDataSource() datasource.DataSource {
	return &PublicKeyDataSource{}
}

func (d *PublicKeyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_public_key"
}

func (d *PublicKeyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
Hold a publicKey from an identity, created on an other Device (private key is unknown)

As an example: 

A team gives his owner a public key to add them to the cryptvault. 

This can be managed by owner over this data source.
		`,

		Attributes: map[string]schema.Attribute{
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public Key of identity",
				Description:         "Public Key of identity",
				Required:            true,
			},
		},
	}
}

func (d *PublicKeyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PublicKeyDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := helper.GetPublicKeyFromB64String(data.PublicKey.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Public key is invalid can not be decoded", err.Error())
		return
	}

	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
