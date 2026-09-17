---
page_title: sks_nodepool migration guide
description: |-
    migrating sks_nodepool resources from provider version ~> 0.72.x to ~> 0.73.x
---

# Migrating SKS Node Pool from v0.72.x to v0.73.x

This guide covers the migration of `exoscale_sks_nodepool` (resource and data source) from provider version ~> 0.72.x to ~> 0.73.x.

## Overview

Version 0.73.0 migrates `exoscale_sks_nodepool` (resource and data source) from the legacy SDKv2 implementation to the Terraform plugin framework, continuing the migration started with `exoscale_sks_cluster` in a previous release.

As part of this migration, the `kubelet_image_gc` argument changes from a repeatable block to an attribute assignment, so that it can properly reflect the cluster's effective kubelet image garbage collection policy (including platform defaults) instead of only echoing back what you explicitly configured.

Your Terraform state is upgraded automatically the next time you run `plan` or `apply` — no `terraform state` surgery, and no resources are recreated.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### `kubelet_image_gc` Syntax

`kubelet_image_gc` now uses attribute assignment syntax (`=`) with an object, instead of nested block syntax.

**Before (v0.72.x):**
```hcl
resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  # ...

  kubelet_image_gc {
    min_age        = "1h"
    high_threshold = 85
    low_threshold  = 80
  }
}
```

**After (v0.73.x):**
```hcl
resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  # ...

  kubelet_image_gc = {
    min_age        = "1h"
    high_threshold = 85
    low_threshold  = 80
  }
}
```

If your configuration doesn't set `kubelet_image_gc`, no change is required: the attribute is optional either way.

## Migration Steps

### 1. Update Provider Version

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "~> 0.73.0"
    }
  }
}
```

### 2. Update `kubelet_image_gc` Syntax

For every `exoscale_sks_nodepool` resource that sets `kubelet_image_gc`, add an equals sign (`=`) after the argument name, as shown above.

### 3. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider.
2. Run `terraform apply -refresh-only` once. Terraform silently upgrades the on-disk/remote state of every `exoscale_sks_nodepool` resource to the new schema as part of this refresh.
3. Run `terraform plan` to verify there are no further changes.

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_sks_nodepool Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/sks_nodepool)
- [exoscale_sks_nodepool Data Source](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/data-sources/sks_nodepool)
