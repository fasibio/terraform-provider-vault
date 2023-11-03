data "cryptvault_cloud_value" "value1" {
  creator_key = data.cryptvault_cloud_identity.operator.private_key
  vault_id    = data.cryptvault_cloud_vault.my_vault.id
  name        = "VALUES.some.path.value1.name"
}
