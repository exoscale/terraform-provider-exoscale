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
provider "exoscale" {
  version = "~> 0.18.2"
  key     = "EXOxxxxxxxxxxxxxxxxxxxxxxxx"
  secret  = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"

  timeout = 120
}
```

Starting from Terraform 0.13.x:

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "0.18.2"
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
variable "exoscale_api_key" { type = string }
variable "exoscale_api_secret" { type = string }

terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "0.18.2"
    }
  }
}

provider "exoscale" {
  key    = var.exoscale_api_key
  secret = var.exoscale_api_secret
}

locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "my-server" {
  zone        = local.zone
  name        = "my-server"
  type        = "standard.medium"
  template_id = data.exoscale_compute_template.ubuntu.id
  disk_size   = 50
  ssh_key     = "alice"
  user_data   = <<EOF
#cloud-config
package_upgrade: true
EOF
}
```

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding exoscale/exoscale versions matching "0.31.0"...
- Installing exoscale/exoscale v0.31.0...
- Installed exoscale/exoscale v0.31.0 (signed by a HashiCorp partner, key ID XXXXXXXXXXXXXXXX)

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
      + name              = "my-server"
      + public_ip_address = (known after apply)
      + ssh_key           = "alice"
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


[exo-iam]: https://community.exoscale.com/documentation/iam/quick-start/
[tf-doc-provider]: https://www.terraform.io/docs/configuration/providers.html
[tf-exo-gh-examples]: https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples
