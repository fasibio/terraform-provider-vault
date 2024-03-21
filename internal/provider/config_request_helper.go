package provider

import (
	"errors"

	client "github.com/cryptvault-cloud/api"
	"github.com/cryptvault-cloud/helper"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

func getClientRessource(req *resource.ConfigureRequest) (*client.Api, error) {
	client, ok := req.ProviderData.(*client.Api)
	if !ok {
		return nil, errors.New("ProviderData is not *client.Api")
	}
	return client, nil
}

func getClient(req *datasource.ConfigureRequest) (*client.Api, error) {
	client, ok := req.ProviderData.(*client.Api)
	if !ok {
		return nil, errors.New("ProviderData is not *client.Api")
	}
	return client, nil
}

func getProtectedApi(api *client.Api, privateKey basetypes.StringValue, vaultID basetypes.StringValue) (client.ProtectedApiHandler, error) {
	if privateKey.IsNull() {
		return nil, errors.New("private not allowed to be null")
	}
	private_key, err := helper.GetPrivateKeyFromB64String(privateKey.ValueString())
	if err != nil {
		return nil, errors.Join(errors.New("private key is not an ecdsa.Private key"), err)
	}
	if vaultID.IsNull() {
		return nil, errors.New("vaultid not allowed to be null")
	}
	vault_id := vaultID.ValueString()
	return api.GetProtectedApi(private_key, vault_id), nil
}
