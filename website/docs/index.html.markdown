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

resource "exoscale_compute" "my-server" {
  zone         = local.zone
  display_name = "my-server"
  size         = "Small"
  template_id  = data.exoscale_compute_template.ubuntu.id
  disk_size    = 50
  key_pair     = "alice"
  user_data    = <<EOF
#cloud-config
package_upgrade: true
EOF
}
```

```console
$ terraform init

Initializing the backend...

Initializing provider plugins...
- Finding exoscale/exoscale versions matching "0.18.2"...
- Installing exoscale/exoscale v0.18.2...
- Installed exoscale/exoscale v0.18.2 (signed by a HashiCorp partner, key ID 8B58C61D4FFE0C86)

...

$ terraform apply \
    -var exoscale_api_key=$EXOSCALE_API_KEY \
    -var exoscale_api_secret=$EXOSCALE_API_SECRET

data.exoscale_compute_template.ubuntu: Refreshing state...

An execution plan has been generated and is shown below.
Resource actions are indicated with the following symbols:
  + create

Terraform will perform the following actions:

  # exoscale_compute.my-server will be created
  + resource "exoscale_compute" "my-server" {
      + affinity_group_ids = (known after apply)
      + affinity_groups    = (known after apply)
      + disk_size          = 50
      + display_name       = "my-server"
      + gateway            = (known after apply)
      + hostname           = (known after apply)
      + id                 = (known after apply)
      + ip4                = true
      + ip6                = false
      + ip6_address        = (known after apply)
      + ip6_cidr           = (known after apply)
      + ip_address         = (known after apply)
      + key_pair           = "alice"
      + name               = (known after apply)
      + password           = (sensitive value)
      + security_group_ids = (known after apply)
      + security_groups    = (known after apply)
      + size               = "Small"
      + state              = (known after apply)
      + tags               = (known after apply)
      + template           = (known after apply)
      + template_id        = "c19542b7-d269-4bd4-bf7c-2cae36d066d3"
      + user_data          = <<~EOT
            #cloud-config
            package_upgrade: true
        EOT
      + user_data_base64   = (known after apply)
      + username           = (known after apply)
      + zone               = "ch-gva-2"
    }

Plan: 1 to add, 0 to change, 0 to destroy.

Do you want to perform these actions?
...
```

Additional documentation can be found in the [examples][tf-exo-gh-examples]
directory of the source code.


[exo-iam]: https://community.exoscale.com/documentation/iam/quick-start/
[tf-doc-provider]: https://www.terraform.io/docs/configuration/providers.html
[tf-exo-gh-examples]: https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples
