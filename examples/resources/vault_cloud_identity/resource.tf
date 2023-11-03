resource "cryptvault_cloud_identity" "writer" {
  name        = "writer"
  vault_id    = cryptvault_cloud_vault.my_vault.id
  creator_key = cryptvault_cloud_vault.my_vault.operator_private_key
  rights = [
    {
      right_value_pattern = "(rwd)VALUES.some.path.>"
    },
    {
      right_value_pattern = "(rwd)VALUES.some.other.path.>"
    }
  ]
}

resource "cryptvault_cloud_identity" "value1-reader" {
  name        = "reader"
  vault_id    = cryptvault_cloud_vault.my_vault.id
  creator_key = cryptvault_cloud_vault.my_vault.operator_private_key
  rights = [
    {
      right_value_pattern = "(r)VALUES.some.path.value1.*"
    }
  ]
}
