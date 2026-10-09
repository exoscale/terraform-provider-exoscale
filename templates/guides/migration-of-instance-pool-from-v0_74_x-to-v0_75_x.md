---
page_title: instance_pool migration guide
description: |-
    migrating instance_pool resources from provider version ~> 0.74.x to ~> 0.75.x
---

# Migrating Instance Pool from v0.74.x to v0.75.x

This guide covers the migration of the `exoscale_instance_pool` resource from provider version ~> 0.74.x to ~> 0.75.x.

## Overview

Version 0.75.0 migrates the `exoscale_instance_pool` resource from the legacy SDKv2 implementation to the Terraform plugin framework.

As part of this migration, the `service_offering` argument, deprecated since v0.27.0, is removed: `instance_type` is now required. The `instances` and `virtual_machines` attributes become read-only.

Your Terraform state is upgraded automatically the next time you run `plan` or `apply` — no `terraform state` surgery, and no resources are recreated.

The `exoscale_instance_pool` and `exoscale_instance_pool_list` data sources are not affected.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### `service_offering` Removed, `instance_type` Required

`service_offering` only held the size of the instance type, the family being implicitly `standard`. Replace it with `instance_type`, in the `<family>.<size>` format. The [Exoscale CLI](https://github.com/exoscale/cli/) lists the available instance types:

```console
exo compute instance-type list
```

**Before (v0.74.x):**
```hcl
resource "exoscale_instance_pool" "my_instance_pool" {
  # ...

  service_offering = "medium"
}
```

**After (v0.75.x):**
```hcl
resource "exoscale_instance_pool" "my_instance_pool" {
  # ...

  instance_type = "standard.medium"
}
```

If you are not sure of the current type of a pool, `terraform state show` prints it as `instance_type`. References to `exoscale_instance_pool.<name>.service_offering` must be replaced as well, e.g. with `split(".", exoscale_instance_pool.<name>.instance_type)[1]`.

### `instances` and `virtual_machines` are Read-Only

`instances` and `virtual_machines` are computed by the provider: the values set in the configuration were ignored. They can no longer be set, remove any `instances { ... }` block or `virtual_machines` argument from your configuration.

**Before (v0.74.x):**
```hcl
resource "exoscale_instance_pool" "my_instance_pool" {
  # ...

  instances {
    name = "my-instance"
  }
}
```

**After (v0.75.x):**
```hcl
resource "exoscale_instance_pool" "my_instance_pool" {
  # ...
}
```

References to their values are unchanged, e.g. `exoscale_instance_pool.my_instance_pool.instances[*].public_ip_address`. `virtual_machines` remains deprecated in favour of `instances[*].id`.

### Removing `description`, `labels` or `user_data`

Removing `description`, `labels` or `user_data` from the configuration now clears them on the Instance Pool; it used to have no effect. If your configuration doesn't set one of them but the Instance Pool has it, the next plan shows its removal: add it back to the configuration to keep it.

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

### 2. Update Your Configuration

For every `exoscale_instance_pool` resource:

1. Replace `service_offering = "<size>"` with `instance_type = "standard.<size>"`, as shown above.
2. Remove any `instances` block and `virtual_machines` argument.

### 3. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider.
2. Run `terraform apply -refresh-only` once. Terraform silently upgrades the on-disk/remote state of every `exoscale_instance_pool` resource to the new schema as part of this refresh.
3. Run `terraform plan` to verify there are no further changes.

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_instance_pool Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/instance_pool)
