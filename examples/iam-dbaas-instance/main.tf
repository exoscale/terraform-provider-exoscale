# Providers
# -> providers.tf

resource "exoscale_iam_role" "my_api_role" {
  name        = "my-api-role"
  description = "Role that allows only deploying private instances."
  editable    = true

  policy = {
    default_service_strategy = "deny"
    services = {
      dbaas = {
        type = "rules"
        rules = [
          {
            expression = "resources.dbaas_service.name != 'my-dbaas-instance'"
            action     = "deny"
          },
          {
            expression = "true"
            action     = "allow"
          }
        ]
      }
    }
  }
}

resource "exoscale_iam_api_key" "my_api_key" {
  name    = "my-api-key"
  role_id = exoscale_iam_role.my_api_role.id
}

# Outputs
output "my_api_key" {
  value = exoscale_iam_api_key.my_api_key.key
}
output "my_api_secret" {
  value     = exoscale_iam_api_key.my_api_key.secret
  sensitive = true
}
