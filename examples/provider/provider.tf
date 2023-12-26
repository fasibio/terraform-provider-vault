terraform {
  required_providers {
    cryptvault = {
      source = "cryptvault-cloud/cryptvault"
    }
  }
}

provider "cryptvault" {}

resource "cryptvault_cloud_vault" "my_vault" {
  name  = "name_of_vault"
  token = "token_allow_you_to_create_vault"
}

resource "cryptvault_cloud_keypair" "writer" {}
resource "cryptvault_cloud_identity" "writer" {
  name        = "writer"
  vault_id    = cryptvault_cloud_vault.my_vault.id
  creator_key = cryptvault_cloud_vault.my_vault.operator_private_key
  public_key  = cryptvault_cloud_keypair.writer.public_key
  rights = [
    {
      right_value_pattern = "(rwd)VALUES.some.path.>"
    },
    {
      right_value_pattern = "(rwd)VALUES.some.other.path.>"
    }
  ]
}

resource "cryptvault_cloud_keypair" "value1-reader" {}

resource "cryptvault_cloud_identity" "value1-reader" {
  name        = "reader"
  vault_id    = cryptvault_cloud_vault.my_vault.id
  creator_key = cryptvault_cloud_vault.my_vault.operator_private_key
  public_key  = cryptvault_cloud_keypair.value1-reader.public_key
  rights = [
    {
      right_value_pattern = "(r)VALUES.some.path.value1.*"
    }
  ]
}

resource "cryptvault_cloud_value" "value1" {
  vault_id    = cryptvault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value1.name"
  passframe   = "test"
  type        = "String"
  creator_key = cryptvault_cloud_identity.writer.private_key
  depends_on  = [cryptvault_cloud_keypair.writer]
}

resource "cryptvault_cloud_value" "value2" {
  vault_id    = vault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value2.name"
  passframe   = "{\"a\":123}"
  type        = "JSON"
  creator_key = cryptvault_cloud_identity.writer.private_key
  depends_on  = [cryptvault_cloud_keypair.writer]
}


