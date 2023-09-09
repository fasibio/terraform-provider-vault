terraform {
  required_providers {
    vault = {
      source = "fasibio/vault"
    }
  }
}

provider "vault" {}

resource "vault_cloud_vault" "my_vault" {
  name  = "name_of_vault"
  token = "token_allow_you_to_create_vault"
}

resource "vault_cloud_identity" "writer" {
  name        = "writer"
  vault_id    = vault_cloud_vault.my_vault.id
  creator_key = vault_cloud_vault.my_vault.operator_private_key
  rights = [
    {
      right_value_pattern = "(rwd)VALUES.some.path.>"
    },
    {
      right_value_pattern = "(rwd)VALUES.some.other.path.>"
    }
  ]
}

resource "vault_cloud_identity" "value1-reader" {
  name        = "reader"
  vault_id    = vault_cloud_vault.my_vault.id
  creator_key = vault_cloud_vault.my_vault.operator_private_key
  rights = [
    {
      right_value_pattern = "(r)VALUES.some.path.value1.*"
    }
  ]
}

resource "vault_cloud_value" "value1" {
  vault_id    = vault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value1.name"
  passframe   = "test"
  type        = "String"
  creator_key = vault_cloud_identity.writer.private_key
}

resource "vault_cloud_value" "value2" {
  vault_id    = vault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value2.name"
  passframe   = "{\"a\":123}"
  type        = "JSON"
  creator_key = vault_cloud_identity.writer.private_key
}


