terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }
provider "aws" {
  access_key = var.exoscale_api_key
  secret_key = var.exoscale_api_secret

  region = local.my_zone
  endpoints {
    s3 = "https://sos-${local.my_zone}.exo.io"
  }

  # Disable AWS-specific features
  skip_credentials_validation = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
}
