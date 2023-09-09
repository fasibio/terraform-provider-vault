// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	client "github.com/fasibio/vaultapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var valueTypeRegex *regexp.Regexp
var valueTypeStrRegex string

func init() {

	d := []string{"String", "JSON"}
	valueTypeStrRegex = fmt.Sprintf("^(%s)$", strings.Join(d, "|"))
	valueTypeRegex = regexp.MustCompile(valueTypeStrRegex)
}

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ValueResource{}
var _ resource.ResourceWithImportState = &ValueResource{}

func NewValueResource() resource.Resource {
	return &ValueResource{}
}

// ValueResource defines the resource implementation.
type ValueResource struct {
	client *client.Api
}

// ExampleResourceModel describes the resource data model.
type ValueResourceModel struct {
	Id          types.String `tfsdk:"id"`
	VaultID     types.String `tfsdk:"vault_id"`
	LastUpdated types.String `tfsdk:"last_updated"`
	Name        types.String `tfsdk:"name"`
	Passframe   types.String `tfsdk:"passframe"`
	Type        types.String `tfsdk:"type"`
	CreatorKey  types.String `tfsdk:"creator_key"`
}

func (r *ValueResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value"
}

func (r *ValueResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Value id",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "id of related vault",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
				// PlanModifiers: []planmodifier.String{
				// 	stringplanmodifier.UseStateForUnknown(),
				// },
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "key of related value f.e.: VALUES.foo.bar",
				Validators: []validator.String{
					stringvalidator.RegexMatches(client.ValuesPatternRegex, "Have to match value string pattern"),
				},
			},
			"passframe": schema.StringAttribute{
				MarkdownDescription: "passframe of value",
				Required:            true,
				Sensitive:           true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "passframe of value",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(valueTypeRegex, "Have to match "+valueTypeStrRegex),
				},
			},
			"creator_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Private key of identity with rights to create new identities",
			},
		},
	}
}

func (r *ValueResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ValueResourceModel

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
		resp.Diagnostics.AddError("Name is required for creating a new Value", "")
		return
	}

	pApi, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("error creating protectedAPI", err.Error())
		return
	}
	valueId, err := pApi.AddValue(data.Name.ValueString(), data.Passframe.ValueString(), client.ValueType(data.Type.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("error add value", err.Error())
		return
	}
	data.Id = types.StringValue(valueId)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ValueResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pApi, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}
	valueData, err := pApi.GetValueById(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not read value", err.Error())
		return
	}

	data.Id = types.StringValue(valueData.Id)
	err = pApi.SyncValue(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("errory by sync Values", err.Error())
	}
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ValueResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	pApi, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}
	value, err := pApi.UpdateValue(data.Id.ValueString(), data.Name.ValueString(), data.Passframe.ValueString(), client.ValueType(data.Type.ValueString()))
	if err != nil {
		if err != nil {
			resp.Diagnostics.AddError("error by update vault", err.Error())
			return
		}
	}

	data.Id = types.StringValue(value)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ValueResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	pApi, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}

	err = pApi.DeleteValue(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete Value", err.Error())
		return
	}
}

func (r *ValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
