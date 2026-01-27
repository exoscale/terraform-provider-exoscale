---
page_title: dbaas migration guide
description: |-
    migrating dbaas resources from provider version ~> 0.67.x to ~> 0.68.x
---

# Migrating DBAAS from v0.67.x to v0.68.x

This guide covers the migration of `exoscale_dbaas` resources from provider version ~> 0.67.x to ~> 0.68.x.

## Overview

Version 0.68.0 introduces a syntax change for database type-specific configuration blocks in the `exoscale_dbaas` resource. The configuration blocks now use attribute assignment syntax (`=`) instead of nested block syntax.

This is purely a configuration syntax change. No Terraform state modifications nor resource recreations are required.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

## What has Changed

### Database Type Configuration Syntax

All database type-specific configuration blocks (`pg`, `mysql`, `kafka`, `opensearch`, `valkey`, `grafana`) now use attribute assignment syntax with an object.

**Before (v0.67.x):**
```hcl
pg {
  version = "16"
  backup_schedule = "05:00"
}
```

**After (v0.68.x):**
```hcl
pg = {
  version = "16"
  backup_schedule = "05:00"
}
```

## Migration Steps

### 1. Update Provider Version

Update your Terraform configuration to use version 0.68.0 or later:

```hcl
terraform {
  required_providers {
    exoscale = {
      source  = "exoscale/exoscale"
      version = "~> 0.68.0"
    }
  }
}
```

### 2. Update Configuration Syntax

Update your `exoscale_dbaas` resources by adding an equals sign (`=`) after the database type block name.

#### Example: PostgreSQL

**Before:**
```hcl
resource "exoscale_dbaas" "postgres" {
  zone = "ch-gva-2"
  name = "my-postgres-db"
  type = "pg"

  plan = "hobbyist-2"

  maintenance_dow  = "sunday"
  maintenance_time = "02:00:00"

  termination_protection = false

  pg {
    version = "16"
    backup_schedule = "05:00"
  }
}
```

**After:**
```hcl
resource "exoscale_dbaas" "postgres" {
  zone = "ch-gva-2"
  name = "my-postgres-db"
  type = "pg"

  plan = "hobbyist-2"

  maintenance_dow  = "sunday"
  maintenance_time = "02:00:00"

  termination_protection = false

  pg = {
    version = "16"
    backup_schedule = "05:00"
  }
}
```

#### Example: MySQL

**Before:**
```hcl
mysql {
  version = "8"
  backup_schedule = "03:00"
}
```

**After:**
```hcl
mysql = {
  version = "8"
  backup_schedule = "03:00"
}
```

#### Example: Kafka

**Before:**
```hcl
kafka {
  version = "3.6"
  kafka_rest_enabled = true
}
```

**After:**
```hcl
kafka = {
  version = "3.6"
  kafka_rest_enabled = true
}
```

### 3. Verify Changes

After updating your configuration:

1. Run `terraform init -upgrade` to upgrade the provider
2. Run `terraform plan` to verify the changes

You should see output similar to:

```
No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration
and found no differences, so no changes are needed.
```

## Additional Resources
- [exoscale_dbaas Resource](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/dbaas)
