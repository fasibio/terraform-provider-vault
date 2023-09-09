// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"time"

	client "github.com/fasibio/vaultapi"
	"github.com/fasibio/vaulthelper"
	"github.com/fasibio/vaulthelper/helper"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &VaultResource{}
var _ resource.ResourceWithImportState = &VaultResource{}

func NewVaultResource() resource.Resource {
	return &VaultResource{}
}

// VaultResource defines the resource implementation.
type VaultResource struct {
	client *client.Api
}

// ExampleResourceModel describes the resource data model.
type VaultResourceModel struct {
	Id                  types.String `tfsdk:"id"`
	Token               types.String `tfsdk:"token"`
	Name                types.String `tfsdk:"name"`
	LastUpdated         types.String `tfsdk:"last_updated"`
	Operator_Id         types.String `tfsdk:"operator_id"`
	Operator_PublicKey  types.String `tfsdk:"operator_public_key"`
	Operator_PrivateKey types.String `tfsdk:"operator_private_key"`
	Operator_Name       types.String `tfsdk:"operator_name"`
}

func (r *VaultResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vault"
}

func (r *VaultResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Vault id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
				// PlanModifiers: []planmodifier.String{
				// 	stringplanmodifier.UseStateForUnknown(),
				// },
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name for the new vault",
			},
			"token": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Token to verify vault generation is allowed",
			},
			"operator_id": schema.StringAttribute{
				MarkdownDescription: "id of master identity",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"operator_public_key": schema.StringAttribute{
				MarkdownDescription: "Public key of master identity",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"operator_private_key": schema.StringAttribute{
				MarkdownDescription: "Private key of master identity",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"operator_name": schema.StringAttribute{
				MarkdownDescription: "name of master identity",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *VaultResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	client, err := getClientRessource(&req)

	if err != nil {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.: %v", req.ProviderData, err),
		)

		return
	}

	r.client = client
}

func (r *VaultResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VaultResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// If applicable, this is a great opportunity to initialize any necessary
	// provider client data and make a call using it.
	// httpResp, err := r.client.Do(httpReq)
	// if err != nil {
	//     resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create example, got error: %s", err))
	//     return
	// }

	if data.Name.IsNull() {
		resp.Diagnostics.AddError("Name is required for creating a new vault", "")
		return
	}
	privateKey, publicKey, vaultid, err := r.client.NewVault(data.Name.ValueString(), data.Token.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Vault", err.Error())
		return
	}

	data.Id = types.StringValue(vaultid)
	pubKey, err := vaulthelper.NewBase64PublicPem(publicKey)

	if err != nil {
		resp.Diagnostics.AddError("Unable pack public key", err.Error())
		return
	}

	data.Operator_PublicKey = types.StringValue(string(pubKey))
	operatorId, err := pubKey.GetIdentityId(vaultid)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create operatorid", err.Error())
		return
	}
	data.Operator_Id = types.StringValue(operatorId)
	privKey, err := helper.GetB64FromPrivateKey(privateKey)
	if err != nil {
		resp.Diagnostics.AddError("Unable pack private key", err.Error())
		return
	}
	data.Operator_PrivateKey = types.StringValue(privKey)
	pApi, err := getProtectedApi(r.client, data.Operator_PrivateKey, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}

	operator, err := pApi.GetIdentity(data.Operator_Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to get operator", err.Error())
		return
	}
	data.Operator_Name = types.StringValue(*operator.Name)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VaultResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pApi, err := getProtectedApi(r.client, data.Operator_PrivateKey, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}
	vaultData, err := pApi.GetVault()
	if err != nil {
		resp.Diagnostics.AddError("Can not read vault", err.Error())
		return
	}

	data.Id = types.StringValue(vaultData.Id)
	data.Name = types.StringValue(vaultData.Name)

	operator, err := pApi.GetIdentity(data.Operator_Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not read operator identity", err.Error())
		return
	}

	data.Operator_Name = types.StringValue(*operator.Name)
	data.LastUpdated = types.StringValue(vaultData.UpdatedAt.Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VaultResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	pApi, err := getProtectedApi(r.client, data.Operator_PrivateKey, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}
	vault, err := pApi.UpdateVault(data.Name.ValueString())
	if err != nil {
		if err != nil {
			resp.Diagnostics.AddError("error by update vault", err.Error())
			return
		}
	}

	data.Name = types.StringValue(vault.Name)
	data.LastUpdated = types.StringValue(vault.UpdatedAt.Format(time.RFC850))
	data.Id = types.StringValue(vault.Id)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VaultResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VaultResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pApi, err := getProtectedApi(r.client, data.Operator_PrivateKey, data.Id)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}

	err = pApi.DeleteVault(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete vault", err.Error())
		return
	}

}

func (r *VaultResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
