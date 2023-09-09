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
