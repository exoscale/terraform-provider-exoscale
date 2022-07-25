---
layout: "exoscale"
page_title: "Provider: Exoscale"
sidebar-current: "docs-exoscale-index"
description: |-
  The Exoscale provider can be used to manage infrastructure resources running on Exoscale.
---


# Exoscale Provider

## Configuration

The following provider-level settings are supported, either via [HCL
parameters][tf-doc-provider] or environment variables:

* `key` / `EXOSCALE_API_KEY`: Exoscale account API key
* `secret` / `EXOSCALE_API_SECRET`: Exoscale account API secret
* `timeout`: Global async operations waiting time in seconds (default: `300`)

At least an [Exoscale API key and secret][exo-iam] must be provided in order to
use the Exoscale Terraform provider.


### Example

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
    }
  }
}

provider "exoscale" {
  key = "EXOxxxxxxxxxxxxxxxxxxxxxxxx"
  secret = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
}
```


### Fine-tuning Timeout durations

In addition of the global `timeout` provider setting, the waiting time of async
operations can be fine-tuned per resource and per operation type:

```hcl
resource "exoscale_instance_pool" "web" {
  # ...

  timeouts {
    create = "1m"
    read   = "2m"
    update = "3m"
    delete = "4m"
  }
}
```


## Usage

Here is a simple HCL configuration provisioning an Exoscale Compute instance:

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
    }
  }
}

variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }
provider "exoscale" {
  key    = var.exoscale_api_key
  secret = var.exoscale_api_secret
}

locals {
  my_zone     = "ch-gva-2"
  my_template = "Linux Ubuntu 22.04 LTS 64-bit"
  my_ssh_key  = "my-ssh-key"
}

data "exoscale_compute_template" "my_template" {
  zone = local.my_zone
  name = local.my_tempate
}

resource "exoscale_compute_instance" "my_instance" {
  zone        = local.my_zone
  name        = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  ssh_key     = local.my_ssh_key
  user_data   = "#cloud-config\npackage_upgrade: true\n"
}
```

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding latest version of exoscale/exoscale...
- Installing exoscale/exoscale v0.38.0...
- Installed exoscale/exoscale v0.38.0 (signed by a HashiCorp partner, key ID 81426F034A3D05F7)

...

$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # exoscale_compute_instance.my-server will be created
  + resource "exoscale_compute_instance" "my-server" {
      + created_at        = (known after apply)
      + disk_size         = 50
      + id                = (known after apply)
      + ipv6              = false
      + ipv6_address      = (known after apply)
      + name              = "my-instance"
      + public_ip_address = (known after apply)
      + ssh_key           = "my-ssh-key"
      + state             = (known after apply)
      + template_id       = "3ebca0c5-63f4-4055-b325-3cef0e68fa98"
      + type              = "standard.medium"
      + user_data         = <<-EOT
            #cloud-config
            package_upgrade: true
        EOT
      + zone              = "ch-gva-2"
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Do you want to perform these actions?
  Terraform will perform the actions described above.
  Only 'yes' will be accepted to approve.

  Enter a value:
...
```

Additional documentation can be found in the [examples][tf-exo-gh-examples]
directory of the source code.

## Simple Object Storage (SOS)

-> The Exoscale provider does not manage [SOS][exo-sos] resources. As SOS is
S3-compatible, [Terraform AWS provider][tf-provider-aws] can be used instead to
[manage your SOS resources][exo-sos-terraform].

[exo-iam]: https://community.exoscale.com/documentation/iam/quick-start/
[tf-doc-provider]: https://www.terraform.io/docs/configuration/providers.html
[tf-exo-gh-examples]: https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples
[tf-provider-aws]: https://registry.terraform.io/providers/hashicorp/aws/latest/docs
[exo-sos]: https://community.exoscale.com/documentation/storage
[exo-sos-terraform]: https://community.exoscale.com/documentation/storage/terraform/
