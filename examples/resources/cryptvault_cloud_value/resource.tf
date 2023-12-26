resource "cryptvault_cloud_value" "value1" {
  vault_id    = cryptvault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value1.name"
  passframe   = "test"
  type        = "String"
  creator_key = cryptvault_cloud_identity.writer.private_key
  depends_on  = [cryptvault_cloud_keypair.writer]

}

resource "cryptvault_cloud_value" "value2" {
  vault_id    = cryptvault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value2.name"
  passframe   = "1234AVT"
  type        = "String"
  creator_key = cryptvault_cloud_identity.writer.private_key
  depends_on  = [cryptvault_cloud_keypair.writer]

}


