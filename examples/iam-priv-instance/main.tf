# Providers
# -> providers.tf

resource "exoscale_iam_role" "my_api_role" {
  name        = "my-api-role"
  description = "Role that allows only deploying private instances."
  editable    = true

  policy = {
    default_service_strategy = "allow"
    services = {
      compute = {
        type = "rules"
        rules = [
          {
            expression = "operation == 'create-instance' && (!parameters.has('public_ip_assignment') || parameters.public_ip_assignment != 'none')"
            action     = "deny"
          },
          {
            expression = "true"
            action = "allow"
          }
        ]
      }
      "compute-legacy" = {
        type = "deny"
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
