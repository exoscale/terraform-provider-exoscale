provider "exoscale" {
  version = "~> 0.15"
  key     = var.key
  secret  = var.secret
}


terraform {
  backend "s3" {
    bucket   = "example-provisioning-bucket"
    key      = "terraform.tfstate"
    region   = "ch-gva-2"
    endpoint = "https://sos-ch-gva-2.exo.io"

    # Skip AWS-specific local config validation
    # https://www.terraform.io/docs/backends/types/s3.html#skip_credentials_validation
    #
    # Skip AWS IAM validation
    skip_credentials_validation = true
    # Skip AWS region validation
    skip_region_validation = true
  }
}
