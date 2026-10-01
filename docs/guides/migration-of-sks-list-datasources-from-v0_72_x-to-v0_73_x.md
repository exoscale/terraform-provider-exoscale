---
page_title: sks list data sources migration guide
description: |-
    migrating exoscale_sks_cluster_list and exoscale_sks_nodepool_list data sources from provider version ~> 0.72.x to ~> 0.73.x
---

# Migrating SKS List Data Sources from v0.72.x to v0.73.x

This guide covers the migration of the `exoscale_sks_cluster_list` and `exoscale_sks_nodepool_list` data sources from provider version ~> 0.72.x to ~> 0.73.x.

## Overview

Version 0.73.0 migrates `exoscale_sks_cluster_list` and `exoscale_sks_nodepool_list` from the legacy SDKv2 implementation to the Terraform plugin framework, continuing the SKS migration started with `exoscale_sks_cluster`, `exoscale_sks_nodepool` and `exoscale_sks_kubeconfig` in a previous release.

As part of this migration, both data sources drop their generic, per-attribute filter arguments: `zone` is now the only input they accept, and they always return every cluster / node pool found in that zone. If you relied on server-side filtering, filter the returned list yourself with a Terraform `for` expression instead. A handful of attributes on the returned list items were also removed, and a few on `exoscale_sks_nodepool_list` are now populated correctly for the first time — see below.

Since these are data sources, there is no resource state to carry forward or upgrade: they are fully re-read on every `plan`/`apply`. The breaking change is entirely in your configuration — the arguments you pass in, and the attributes you reference from the result — not in any stored state.

~> **Note:** Before migrating your configuration you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### Filtering is no longer built in

Previously, every attribute of a cluster/node pool doubled as an optional top-level filter argument on the list data source (with regex support via `/.../`). That's gone: both data sources now only accept `zone` (and `timeouts`) as input, and always return the full list for that zone.

**Before (v0.72.x):**
```hcl
data "exoscale_sks_cluster_list" "prod" {
  zone = "de-fra-1"
  labels = {
    "customer" = "/.*telecom.*/"
  }
}
```

**After (v0.73.x):**
```hcl
data "exoscale_sks_cluster_list" "all" {
  zone = "de-fra-1"
}

locals {
  prod_clusters = [
    for c in data.exoscale_sks_cluster_list.all.clusters :
    c if can(regex(".*telecom.*", lookup(c.labels, "customer", "")))
  ]
}
```

The same applies to `exoscale_sks_nodepool_list`:

**Before (v0.72.x):**
```hcl
data "exoscale_sks_nodepool_list" "workers" {
  zone = "de-fra-1"
  name = "/.*-workers-.*/"
}
```

**After (v0.73.x):**
```hcl
data "exoscale_sks_nodepool_list" "all" {
  zone = "de-fra-1"
}

locals {
  worker_nodepools = [
    for np in data.exoscale_sks_nodepool_list.all.nodepools :
    np if can(regex(".*-workers-.*", np.name))
  ]
}
```

### Removed attributes on `exoscale_sks_cluster_list` items

The following attributes have been removed from each entry of `clusters`: `aggregation_ca`, `audit`, `control_plane_ca`, `create_default_security_group`, `enable_karpenter`, `exoscale_ccm`, `exoscale_csi`, `kubelet_ca`, `metrics_server`, `oidc`, and the per-item `zone` (redundant with the data source's own `zone` argument). These attributes were declared in the legacy schema but never actually populated by the provider, so they always read as an empty or zero value — no real data is lost. If you need any of them, look up the specific cluster with the [exoscale_sks_cluster](../data-sources/sks_cluster.md) data source instead.

### `exoscale_sks_nodepool_list` items: one removed, some fixed, one new

The per-item `zone` attribute was removed, same as above (redundant with the data source's own `zone` argument).

`storage_lvm`, `ipv6`, `template_id` and `kubelet_image_gc` are now populated from each node pool's actual state. Previously they were declared in the schema but never set by the provider, so they always read as `false`/empty regardless of the real node pool configuration — if your configuration depended on that always-empty behavior, double check it against the real values.

A new `cluster_id` attribute is now populated on each item, identifying which `exoscale_sks_cluster` the node pool belongs to. This existed in the legacy schema too, but only as an input-only filter argument that could never actually match anything (it was never compared against real node pool data), so it silently returned an empty list whenever used. It's now a real, always-populated output attribute.

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

### 2. Replace filter arguments with a `for` expression

For every `exoscale_sks_cluster_list` or `exoscale_sks_nodepool_list` data source that sets an argument other than `zone` (or `timeouts`), remove it and instead filter the `clusters`/`nodepools` list yourself, as shown above.

### 3. Remove references to dropped attributes

Search your configuration for the removed `clusters[*]` attributes listed above (`aggregation_ca`, `audit`, `control_plane_ca`, `create_default_security_group`, `enable_karpenter`, `exoscale_ccm`, `exoscale_csi`, `kubelet_ca`, `metrics_server`, `oidc`) and the per-item `zone` on both data sources, and remove them.

### 4. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider.
2. Run `terraform plan` to verify there are no further changes beyond what you expect from the newly-populated `exoscale_sks_nodepool_list` attributes.

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_sks_cluster_list Data Source](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/data-sources/sks_cluster_list)
- [exoscale_sks_nodepool_list Data Source](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/data-sources/sks_nodepool_list)
