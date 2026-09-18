terraform {
  required_providers {
    exoscale = {
      source = "exoscale/exoscale"
    }
  }
}

variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }
provider "exoscale" {
  key    = var.exoscale_api_key
  secret = var.exoscale_api_secret
}
