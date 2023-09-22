// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	client "github.com/cryptvault-cloud/api"
	"github.com/cryptvault-cloud/helper"
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

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IdentityResource{}
var _ resource.ResourceWithImportState = &IdentityResource{}

var ValuePatternRegex *regexp.Regexp
var ValuesPatternRegex *regexp.Regexp

const ValuePatternRegexStr = `^\((?P<directions>(r|w|d)+)\)(?P<target>(VALUES|IDENTITY|SYSTEM))(?P<pattern>(\.[a-z0-9_\->\*]+)+)$`
const ValuesPatternRegexStr = `^(VALUES|IDENTITY|SYSTEM)(\.[a-z0-9_\-]+)+$`

func init() {
	ValuePatternRegex = regexp.MustCompile(ValuePatternRegexStr)
	ValuesPatternRegex = regexp.MustCompile(ValuesPatternRegexStr)
}

func NewIdentityResource() resource.Resource {
	return &IdentityResource{}
}

// IdentityResource defines the resource implementation.
type IdentityResource struct {
	client *client.Api
}

// ExampleResourceModel describes the resource data model.
type IdentityResourceModel struct {
	Id          types.String          `tfsdk:"id"`
	Name        types.String          `tfsdk:"name"`
	LastUpdated types.String          `tfsdk:"last_updated"`
	PublicKey   types.String          `tfsdk:"public_key"`
	PrivateKey  types.String          `tfsdk:"private_key"`
	VaultID     types.String          `tfsdk:"vault_id"`
	CreatorKey  types.String          `tfsdk:"creator_key"`
	Rights      []RightsResourceModel `tfsdk:"rights"`
}

type RightsResourceModel struct {
	RightValuePattern types.String `tfsdk:"right_value_pattern"`
}

func (r *IdentityResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity"
}

func (r *IdentityResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// This description is used by the documentation generator and the language server.
		MarkdownDescription: "Example resource",

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Identity id",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Name for the new Identity",
			},
			"last_updated": schema.StringAttribute{
				Computed: true,
				// PlanModifiers: []planmodifier.String{
				// 	stringplanmodifier.UseStateForUnknown(),
				// },
			},
			"public_key": schema.StringAttribute{
				MarkdownDescription: "Public key of identity",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"private_key": schema.StringAttribute{
				MarkdownDescription: "Private key of identity",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vault_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Vault id",
			},
			"creator_key": schema.StringAttribute{
				Required:            true,
				Sensitive:           true,
				MarkdownDescription: "Private key of identity with rights to create new identities",
			},
			"rights": schema.ListNestedAttribute{
				Required:            true,
				MarkdownDescription: "Permissions for this new Identity",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"right_value_pattern": schema.StringAttribute{
							Required: true,
							MarkdownDescription: fmt.Sprintf(`
Path to right point separated. 
						
Have to match /%s/

some examples: 
	- (rwd)VALUES.foo.bar
	- (rdw)VALUES.foo.>
	- (rwd)VALUES.>
	- (w)IDENTITY.>
	- (r)IDENTITY.>
	- (rd)VALUES.foo.*

Explain: 
- r = read
- w = write
- d = delete
- > = same area and deeper (next . split group)
- * = same area but each possible string
							`, ValuePatternRegexStr),
							Validators: []validator.String{
								stringvalidator.RegexMatches(ValuePatternRegex, "Have to match right string pattern"),
							},
						},
					},
				},
			},
		},
	}
}

func (r *IdentityResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdentityResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data IdentityResourceModel

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
		resp.Diagnostics.AddError("Name is required for creating a new Identity", "")
		return
	}

	if data.VaultID.IsNull() {
		resp.Diagnostics.AddError("vault is required for creating a new Identity", "")
		return
	}

	if data.CreatorKey.IsNull() {
		resp.Diagnostics.AddError("Creator private key is required for creating a new Identity", "")
		return
	}

	if len(data.Rights) == 0 {
		resp.Diagnostics.AddError("Minimum one right is required for creating a new Identity", "")
		return
	}

	pAPI, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Error by creating the API", err.Error())
		return
	}

	privateKey, publicKey, err := r.client.GetNewIdentityKeyPair()
	if err != nil {
		resp.Diagnostics.AddError("Error creating new KeyPair", err.Error())
		return
	}

	rightInputs, err := getRightInputs(data.Rights)
	if err != nil {
		resp.Diagnostics.AddError("error by rights convert"+err.Error(), err.Error())
		return
	}

	result, err := pAPI.AddIdentity(data.Name.ValueString(), publicKey, rightInputs)
	if err != nil {
		resp.Diagnostics.AddError("error by creating new indentity", err.Error())
		return
	}
	data.Id = types.StringValue(result.IdentityId)
	pubKeyPem, err := helper.NewBase64PublicPem(publicKey)
	if err != nil {
		resp.Diagnostics.AddError("error by create pem from public key", err.Error())
		return
	}
	data.PublicKey = types.StringValue(string(pubKeyPem))
	privKeyPem, err := helper.GetB64FromPrivateKey(privateKey)
	if err != nil {
		resp.Diagnostics.AddError("error by create pem from private key", err.Error())
		return
	}
	data.PrivateKey = types.StringValue(privKeyPem)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	values, err := pAPI.GetAllRelatedValues(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error by get all related values for current creating identity", err.Error())
		return
	}
	for _, v := range values {
		err := pAPI.SyncValue(v)
		if err != nil {
			resp.Diagnostics.AddError(fmt.Sprintf("error by sync value %s for current creating identity", v), err.Error())
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "created a resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func getRightInputs(rights []RightsResourceModel) ([]*client.RightInput, error) {
	rightInputs := make([]*client.RightInput, 0)
	var errs error = nil
	for _, v := range rights {
		tmp, err := client.GetRightDescriptionByString(v.RightValuePattern.ValueString())
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("error by right %s :%s", v.RightValuePattern.ValueString(), err.Error()))
			continue
		}
		for _, tmpV := range tmp {
			rightInputs = append(rightInputs, &client.RightInput{
				Target:            tmpV.Target,
				Right:             tmpV.Right,
				RightValuePattern: tmpV.RightValue,
			})
		}
	}
	return rightInputs, errs
}

func (r *IdentityResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IdentityResourceModel

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

	if data.Id.IsUnknown() || data.Id.IsNull() || data.Id.ValueString() == "" {
		if data.PublicKey.IsNull() || data.PublicKey.IsUnknown() {
			resp.Diagnostics.AddError("Public key is not set... this schould not happen", "")
			return
		}
		id, err := helper.Base64PublicPem(data.PublicKey.ValueString()).GetIdentityId(data.VaultID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to create id form public key", err.Error())
			return
		}

		data.Id = types.StringValue(id)
	}

	identityData, err := pApi.GetIdentity(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Can not read vault", err.Error())
		return
	}

	data.Id = types.StringValue(identityData.Id)
	data.Name = types.StringValue(*identityData.Name)
	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))
	data.VaultID = types.StringValue(identityData.VaultID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IdentityResourceModel

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
	rightInputs, err := getRightInputs(data.Rights)
	if err != nil {
		resp.Diagnostics.AddError("error by rights convert"+err.Error(), err.Error())
		return
	}

	if data.Id.IsUnknown() || data.Id.IsNull() || data.Id.ValueString() == "" {
		if data.PublicKey.IsNull() || data.PublicKey.IsUnknown() {
			resp.Diagnostics.AddError("Public key is not set... this schould not happen", "")
			return
		}
		id, err := helper.Base64PublicPem(data.PublicKey.ValueString()).GetIdentityId(data.VaultID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to create id form public key", err.Error())
			return
		}
		data.Id = types.StringValue(id)
	}

	_, err = pApi.UpdateIdentity(data.Id.ValueString(), data.Name.ValueString(), rightInputs)
	if err != nil {
		if err != nil {
			resp.Diagnostics.AddError("error by update vault", err.Error())
			return
		}
	}

	data.LastUpdated = types.StringValue(time.Now().Format(time.RFC850))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *IdentityResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IdentityResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	if data.Id.IsUnknown() || data.Id.IsNull() || data.Id.ValueString() == "" {
		if data.PublicKey.IsNull() || data.PublicKey.IsUnknown() {
			resp.Diagnostics.AddError("Public key is not set... this schould not happen", "")
			return
		}
		id, err := helper.Base64PublicPem(data.PublicKey.ValueString()).GetIdentityId(data.VaultID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to create id form public key", err.Error())
			return
		}
		data.Id = types.StringValue(id)
	}

	pApi, err := getProtectedApi(r.client, data.CreatorKey, data.VaultID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to build protected Api", err.Error())
		return
	}

	err = pApi.DeleteIdentity(data.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete vault", err.Error())
		return
	}

}

func (r *IdentityResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
