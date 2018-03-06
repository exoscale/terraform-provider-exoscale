# Terraform provider for Exoscale

[![Build Status](https://travis-ci.org/exoscale/terraform-provider-exoscale.svg?branch=master)](https://travis-ci.org/exoscale/terraform-provider-exoscale)

## Installation

1. Download `terraform-provider-exoscale` from the [releases page](https://github.com/exoscale/terraform-provider-exoscale/releases);
2. Put it into the `.terraform/plugins/(darwin|linux|windows)_amd64` folder and make it executable;
3. Run `terraform init`.

```
$ terraform providers
.
└── provider.exoscale
```

Go read the article on our weblog [Terraform on Exoscale](https://www.exoscale.com/syslog/2016/09/14/terraform-with-exoscale/).


## Resources

The documentation has moved into `website`.

## Storage on S3

```hcl
terraform = {
  backend "s3" {
    bucket = "..."
    endpoint = "https://sos-ch-dk-2.exo.io"
    key = "..."
    region = "ch-dk-2"
    access_key = "..."
    secret_key = "..."

    # Deactivate the AWS specific behaviours
    #
    # https://www.terraform.io/docs/backends/types/s3.html#skip_credentials_validation
    skip_credentials_validation = true
    skip_get_ec2_platforms = true
    skip_requesting_account_id = true
    skip_metadata_api_check = true
    skip_region_validation = true
  }
}
```

## Contributing

Contributions are welcome and we encourage you to build the provider locally
before sending a pull request.

### Building

```
$ git clone https://github.com/exoscale/terraform-provider-exoscale
$ cd terraform-provider-exoscale
$ make build

# making a release (for Exoscale staff only)
$ make release
```

### Development

```
# quick build of the provider
$ make

# updating the dependencies
$ make deps-update
```
