---
page_title: elastic_ip migration guide
description: |-
    migrating elastic_ip resources from provider version ~> 0.74.x to ~> 0.75.x
---

# Migrating Elastic IP from v0.74.x to v0.75.x

This guide covers the migration of `exoscale_elastic_ip` (resource and data source) from provider version ~> 0.74.x to ~> 0.75.x.

## Overview

Version 0.75.0 migrates `exoscale_elastic_ip` (resource and data source) from the legacy SDKv2 implementation to the Terraform plugin framework.

As part of this migration, the `healthcheck` argument changes from a repeatable block to an attribute assignment, since an Elastic IP has at most one healthcheck.

Your Terraform state is upgraded automatically the next time you run `plan` or `apply` — no `terraform state` surgery, and no resources are recreated.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### `healthcheck` Syntax

`healthcheck` now uses attribute assignment syntax (`=`) with an object, instead of nested block syntax.

**Before (v0.74.x):**
```hcl
resource "exoscale_elastic_ip" "my_elastic_ip" {
  # ...

  healthcheck {
    mode         = "https"
    port         = 443
    uri          = "/health"
    interval     = 5
    timeout      = 3
    strikes_ok   = 2
    strikes_fail = 3
    tls_sni      = "example.net"
  }
}
```

**After (v0.75.x):**
```hcl
resource "exoscale_elastic_ip" "my_elastic_ip" {
  # ...

  healthcheck = {
    mode         = "https"
    port         = 443
    uri          = "/health"
    interval     = 5
    timeout      = 3
    strikes_ok   = 2
    strikes_fail = 3
    tls_sni      = "example.net"
  }
}
```

If your configuration doesn't set `healthcheck`, no change is required: the attribute is optional either way.

### References to `healthcheck` Attributes

The healthcheck is no longer a list: drop the `[0]` / `.0` index from references, in the resource as well as in the data source.

**Before (v0.74.x):**
```hcl
output "healthcheck_port" {
  value = data.exoscale_elastic_ip.my_elastic_ip.healthcheck[0].port
}
```

**After (v0.75.x):**
```hcl
output "healthcheck_port" {
  value = data.exoscale_elastic_ip.my_elastic_ip.healthcheck.port
}
```

### Adding or Removing a Healthcheck

An *unmanaged* Elastic IP can not become *managed* and vice versa: adding a `healthcheck` to an existing Elastic IP, or removing it, now plans the replacement of the Elastic IP, which changes its IP address.

~> **Warning:** with v0.74.x, removing the `healthcheck` block from the configuration of a *managed* Elastic IP was silently ignored. If your configuration doesn't set `healthcheck` but the Elastic IP has one, the first plan with v0.75.x shows its replacement: add the `healthcheck` back to the configuration (with its current values, see `terraform state show`) to keep the Elastic IP.

### Removing `description`

Removing `description` from the configuration now clears the description of the Elastic IP; it used to be kept. If your configuration doesn't set `description` but the Elastic IP has one, the next plan shows its removal: add it back to the configuration to keep it.

## Migration Steps

### 1. Update Provider Version

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "~> 0.75.0"
    }
  }
}
```

### 2. Update `healthcheck` Syntax

For every `exoscale_elastic_ip` resource that sets `healthcheck`, add an equals sign (`=`) after the argument name, as shown above, and drop the `[0]` index from the references to its attributes.

### 3. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider.
2. Run `terraform apply -refresh-only` once. Terraform silently upgrades the on-disk/remote state of every `exoscale_elastic_ip` resource to the new schema as part of this refresh.
3. Run `terraform plan` to verify there are no further changes.

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_elastic_ip Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/elastic_ip)
- [exoscale_elastic_ip Data Source](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/data-sources/elastic_ip)
