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
  passframe   = "1234AVT"
  type        = "String"
  creator_key = vault_cloud_identity.writer.private_key
}


