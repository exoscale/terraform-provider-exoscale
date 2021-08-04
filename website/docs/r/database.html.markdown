---
layout: "exoscale"
page_title: "Exoscale: exoscale_database"
sidebar_current: "docs-exoscale-database"
description: |-
  Provides an Exoscale database service resource.
---

# exoscale\_database (BETA)

Provides an Exoscale [DBaaS][dbaas-doc] service resource. This can be used to create, modify, and delete database services.

**Note:** this feature is currently in *beta*, changes to this resource can occur in upcoming releases of the provider.


## Example Usage

```hcl
locals {
  zone = "de-fra-1"
}

resource "exoscale_database" "prod" {
  zone = local.zone
  name = "prod"
  type = "pg"
  plan = "startup-4"
  
  maintenance_dow  = "sunday"
  maintenance_time = "23:00:00"
  
  termination_protection = true
  
  user_config = jsonencode({
    pg_version    = "13"
    backup_hour   = 1
    backup_minute = 0
    ip_filter     = ["194.182.161.182/32"]
    pglookout     = {
      max_failover_replication_time_lag = 60
    }
  })
}

output "database_uri" {
  value = exoscale_database.prod.uri
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the database service into.
* `name` - (Required) The name of the database service.
* `type` - (Required) The type of the database service.
* `plan` - (Required) The plan of the database service.
* `maintenance_dow` - The day of week to perform the automated database service maintenance (accepted values: `never`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`).
* `maintenance_time` - The time of day to perform the automated database service maintenance (format: `HH:MM:SS`)
* `user_config` - The database service specific configuration in JSON format.
* `termination_protection` - The database service protection boolean flag against termination/power-off.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `created_at` - The creation date of the database service.
* `disk_size` - The disk size of the database service.
* `features` - The database service feature flags.
* `metadata` - The database service metadata.
* `node_cpus` - The number of CPUs of the database service.
* `node_memory` - The amount of memory of the database service.
* `nodes` - The number of nodes of the database service.
* `state` - The current state of the database service.
* `state` - The current state of the database service.
* `updated_at` - The date of the latest database service update.
* `uri` - The database service connection URI.


## Import

An existing database service can be imported as a resource by specifying `NAME@ZONE`:

```console
$ terraform import exoscale_database.example my-database@de-fra-1
```


[dbaas-doc]: https://community.exoscale.com/documentation/dbaas/
[zone]: https://www.exoscale.com/datacenters/

