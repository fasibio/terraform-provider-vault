package provider

import (
	"context"
	"fmt"

	client "github.com/cryptvault-cloud/api"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &ValueDataSource{}

func NewValueDataSource() datasource.DataSource {
	return &ValueDataSource{}
}

type ValueDataSource struct {
	client client.ApiHandler
}

type ValueDataSourceModel struct {
	Id         types.String `tfsdk:"id"`
	VaultID    types.String `tfsdk:"vault_id"`
	Name       types.String `tfsdk:"name"`
	Passframe  types.String `tfsdk:"passframe"`
	Type       types.String `tfsdk:"type"`
	CreatorKey types.String `tfsdk:"creator_key"`
}

func (d *ValueDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {

	resp.TypeName = req.ProviderTypeName + "_value"
}

func (d *ValueDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "To Read an already exist resource from vault and encrypt them locally by given creator_key ",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "id of identity",
				Description:         "id of identity",
				Optional:            true,
			},
			"vault_id": schema.StringAttribute{
				MarkdownDescription: "ID of used vault",
				Description:         "ID of used vault",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of identity",
				Description:         "Name of identity",
				Optional:            true,
			},
			"passframe": schema.StringAttribute{
				MarkdownDescription: "passframe of value",
				Description:         "passframe of value",
				Computed:            true,
				Sensitive:           true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Type enum of value",
				Description:         "Type enum of value",
				Computed:            true,
			},
			"creator_key": schema.StringAttribute{
				MarkdownDescription: "Private Key of identity",
				Description:         "Private Key of identity",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *ValueDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ValueDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	pApi, err := getProtectedApi(d.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Error building connection API", err.Error())
		return
	}

	if !data.Id.IsNull() {
		value, err := pApi.GetValueById(data.Id.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Not Possible to getValue by id", err.Error())
			return
		}
		data.Name = types.StringValue(value.Name)
		data.Type = types.StringValue(string(value.Type))
		values := make([]client.EncryptenValue, 0)
		for _, v := range value.GetValue() {
			values = append(values, v)
		}
		passframe, err := pApi.GetDecryptedPassframe(values)
		if err != nil {
			resp.Diagnostics.AddError("Unable to encrypt Value", err.Error())
			return
		}
		data.Passframe = types.StringValue(passframe)
	} else if !data.Name.IsNull() {
		value, err := pApi.GetValueByName(data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Not Possible to getValue by name", err.Error())
			return
		}
		values := make([]client.EncryptenValue, 0)
		for _, v := range value.GetValue() {
			values = append(values, v)
		}
		passframe, err := pApi.GetDecryptedPassframe(values)
		if err != nil {
			resp.Diagnostics.AddError("Unable to encrypt Value", err.Error())
			return
		}
		data.Type = types.StringValue(string(value.Type))
		data.Id = types.StringValue(value.Id)
		data.Passframe = types.StringValue(passframe)
	} else {
		resp.Diagnostics.AddError("Id or Name have to be set ", "Not possible to miss id and name attribute")
		return
	}

	tflog.Trace(ctx, "read a data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
