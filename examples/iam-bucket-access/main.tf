# Providers
# -> providers.tf

resource "exoscale_iam_role" "sos_rw_role" {
  name        = "sos-rw-role"
  description = "Role for Read-Write access for 2 buckets"
  editable    = true

  policy = {
    default_service_strategy = "deny"
    services = {
      sos = {
        type = "rules"
        rules = [
          {
            expression = "!(parameters.bucket in ['my-test-bucket', 'my-other-bucket'])"
            action     = "deny"
          },
          {
            expression = "operation in ['list-sos-buckets-usage', 'list-buckets']"
            action     = "allow"
          },
          {
            expression = "operation in ['create-bucket', 'delete-bucket']"
            action     = "deny"
          },
          {
            expression = "true"
            action     = "allow"
          },
        ]
      }
    }
  }
}

resource "exoscale_iam_role" "sos_ro_role" {
  name        = "sos-ro-role"
  description = "Role for Read-Only access to specific Bucket"
  editable    = true

  policy = {
    default_service_strategy = "deny"
    services = {
      sos = {
        type = "rules"
        rules = [
          {
            expression = "operation in ['list-sos-buckets-usage', 'list-buckets']"
            action     = "allow"
          },
          {
            expression = "!(parameters.bucket == 'my-test-bucket')"
            action     = "deny"
          },
          {
            expression = "operation in ['list-objects', 'get-object']"
            action     = "allow"
          },
          {
            expression = "operation in ['get-bucket-acl', 'get-bucket-cors', 'get-bucket-ownership-controls']"
            action     = "allow"
          }
        ]
      }
    }
  }
}

resource "exoscale_iam_api_key" "sos_rw_key" {
  name    = "sos_rw-api-key"
  role_id = exoscale_iam_role.sos_rw_role.id
}

resource "exoscale_iam_api_key" "sos_ro_key" {
  name    = "sos-ro-api-key"
  role_id = exoscale_iam_role.sos_ro_role.id
}

# Outputs
output "sos_rw_key" {
  value = exoscale_iam_api_key.sos_rw_key.key
}
output "sos_rw_secret" {
  value     = exoscale_iam_api_key.sos_rw_key.secret
  sensitive = true
}
output "sos_ro_key" {
  value = exoscale_iam_api_key.sos_ro_key.key
}
output "sos_ro_secret" {
  value     = exoscale_iam_api_key.sos_ro_key.secret
  sensitive = true
}
