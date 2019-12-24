---
layout: "exoscale"
page_title: "Provider: Exoscale"
sidebar-current: "docs-exoscale-index"
description: |-
  The Exoscale provider is used to interact with the many resources offered by Exoscale.com. The provider needs to be configured with the proper credentials before it can be used.
---

# Exoscale Provider

## Usage

What follows below is the usage instructions for fully utilizing the Exoscale
resource plugin.  Additional documentation can be found in the examples directory.

### Provider requirements

```hcl
provider "exoscale" {
  version = "~> 0.15"
  key = "EXO..."
  secret = "..."

  timeout = 60          # default: waits 60 seconds in total for a resource
  delay = 5             # default: waits 5 seconds between each poll request
}
```

# or

```hcl
provider "exoscale" {
  version = "~> 0.15"

  config = "cloudstack.ini"   # default: filename
  region = "cloudstack"       # default: section name
}
```

You are required to provide at least the API token and secret key in order
to make use of the remaining Terraform resources.

The `timeout` is the maximum amount of time (in seconds, default: `60`) to wait
for async tasks to complete. Currently, this is used during the creation of
`compute` and `anti-affinity` resources.

### `cloudstack.ini`

```ini
[cloudstack]

endpoint = "https://api.exoscale.com/v1"
key = "EXO..."
secret = "..."
```

### Environment variables

You can specify the following keys using those environment variables.

- `key` - `EXOSCALE_KEY`, or `EXOSCALE_API_KEY`;

- `secret` - `EXOSCALE_SECRET`, or `EXOSCALE_API_SECRET`;

- `config` - `EXOSCALE_CONFIG`;

- `region` - `EXOSCALE_REGION`;

- `timeout` - `EXOSCALE_TIMEOUT` global timeout;

- `compute_endpoint` - `EXOSCALE_ENDPOINT`, or `EXOSCALE_COMPUTE_ENDPOINT`;

- `dns_endpoint` - `EXOSCALE_DNS_ENDPOINT`.

## Timeouts

All resources support controlling the waiting time of the four basic operations.

```hcl
resource "exoscale_..." "name" {
  timeouts {
    create = "1m"
    read = "2m"
    update = "3m"
    delete = "4m"
  }
}
```
