data "vault_cloud_identity" "operator" {
  private_key = "long_long_private_key"
  vault_id    = data.vault_cloud_vault.my_vault.id
}
