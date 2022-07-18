terraform {
  required_providers {
    exoscale = {
      source = "exoscale/exoscale"
    }
  }

  backend "s3" {
    bucket   = "example-provisioning-bucket"
    key      = "terraform.tfstate"
    region   = "ch-gva-2"
    endpoint = "https://sos-ch-gva-2.exo.io"

    # Disable AWS-specific features
    skip_credentials_validation = true
    skip_region_validation      = true
  }
}

variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }
provider "exoscale" {
  key    = var.exoscale_api_key
  secret = var.exoscale_api_secret
}
