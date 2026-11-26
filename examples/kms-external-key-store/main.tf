# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone = "ch-gva-2"
}

# XKS proxy you operate, reachable from Exoscale over HTTPS
variable "xks_proxy_endpoint" { type = string } # e.g. https://xks.example.com/kms/xks/v1
variable "xks_proxy_access_key" { type = string }
variable "xks_proxy_secret_key" {
  type      = string
  sensitive = true
}
variable "xks_external_key_id" { type = string } # ID of the key in your key manager

# External Key Store
resource "exoscale_kms_key_store" "my_key_store" {
  zone = local.my_zone
  name = "my-key-store"

  proxy = {
    endpoint              = var.xks_proxy_endpoint
    access_key            = var.xks_proxy_access_key
    secret_key_wo         = var.xks_proxy_secret_key
    secret_key_wo_version = 1
  }

  # KMS keys can only be created in a connected key store.
  connected = true
}

# KMS key backed by the external key
resource "exoscale_kms_key" "my_external_key" {
  zone = local.my_zone
  name = "my-external-key"

  xks = {
    key_store_id    = exoscale_kms_key_store.my_key_store.id
    external_key_id = var.xks_external_key_id
  }

  # Data encrypted with the key can only be decrypted with it: refuse to destroy it.
  lifecycle {
    prevent_destroy = true
  }
}

# Outputs
output "my_external_key_id" {
  value = exoscale_kms_key.my_external_key.id
}
