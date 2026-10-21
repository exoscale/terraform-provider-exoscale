---
page_title: domain migration guide
description: |-
    migrating domain resources from provider version ~> 0.70.x to ~> 0.71.x
---

# Migrating Domain from v0.70.x to v0.71.x

This guide covers the migration of `exoscale_domain` and `exoscale_domain_record` resources (and their data sources) from provider version ~> 0.70.x to ~> 0.71.x.

## Overview

Version 0.71.0 migrates `exoscale_domain` and `exoscale_domain_record` (resources and data sources) from the legacy SDKv2 implementation to the Terraform plugin framework.

No configuration syntax changes are required, and no resources are recreated. The only breaking change is the removal of four attributes from `exoscale_domain` that were already deprecated and always empty.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### Removed attributes on `exoscale_domain`

`token`, `state`, `auto_renew` and `expires_on` have been removed from the `exoscale_domain` resource. These attributes were marked deprecated ("Not used, will be removed in the future") and were never populated by the provider, so no real data is lost. If your configuration references any of them (for example in an `output`, or interpolated into another resource), remove those references.

## Migration Steps

### 1. Update Provider Version

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "~> 0.71.0"
    }
  }
}
```

### 2. Remove references to the dropped attributes

Search your configuration for `.token`, `.state`, `.auto_renew` or `.expires_on` used against an `exoscale_domain` resource or data source, and remove them.

### 3. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider.
2. Run `terraform apply -refresh-only` once.
3. Run `terraform plan` to verify there are no further changes.

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_domain Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/domain)
- [exoscale_domain_record Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/domain_record)
