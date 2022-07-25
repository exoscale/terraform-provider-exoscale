terraform {
  required_providers {
    exoscale = {
      source = "exoscale/exoscale"
    }
    cloudinit = {
      source = "hashicorp/cloudinit"
    }
  }
}

variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }
provider "exoscale" {
  key    = var.exoscale_api_key
  secret = var.exoscale_api_secret
}
