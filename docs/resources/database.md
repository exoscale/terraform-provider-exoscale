---
page_title: "Exoscale: exoscale_database"
description: |-
  Provides an Exoscale database service resource.
---

# exoscale\_database

Provides an Exoscale [DBaaS][dbaas-doc] service resource. This can be used to create, modify, and delete database services.


## Example Usage

```hcl
locals {
  zone = "ch-dk-2"
}

resource "exoscale_database" "pg_prod" {
  zone = local.zone
  name = "pg-prod"
  type = "pg"
  plan = "startup-4"
  
  maintenance_dow  = "sunday"
  maintenance_time = "23:00:00"

  termination_protection = true
  
  pg {
    version = "13"
    backup_schedule = "04:00"

    ip_filter = [
      "1.2.3.4/32",
      "5.6.7.8/32",
    ]
    
    pg_settings = jsonencode({
      timezone: "Europe/Zurich"
    })
  }
}

output "database_uri" {
  value = exoscale_database.pg_prod.uri
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the database service into.
* `name` - (Required) The name of the database service.
* `type` - (Required) The type of the database service (accepted values: `kafka`, `mysql`, `pg`, `redis`).
* `plan` - (Required) The plan of the database service (`exo dbaas type show <TYPE>` for reference).
* `maintenance_dow` - The day of week to perform the automated database service maintenance (accepted values: `never`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`).
* `maintenance_time` - The time of day to perform the automated database service maintenance (format: `HH:MM:SS`)
* `termination_protection` - The database service protection boolean flag against termination/power-off.
* `kafka` - *kafka* database service type specific arguments. Structure is documented below.
* `mysql` - *mysql* database service type specific arguments. Structure is documented below.
* `pg` - *pg* database service type specific arguments. Structure is documented below.
* `redis` - *redis* database service type specific arguments.Structure is documented below.

The `kafka` block supports:

* `enable_cert_auth` - Enable certificate-based authentication method.
* `enable_kafka_connect` - Enable Kafka Connect.
* `enable_kafka_rest` - Enable Kafka REST.
* `enable_sasl_auth` - Enable SASL-based authentication method.
* `enable_schema_registry` - Enable Schema Registry.
* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `kafka_connect_settings` - Kafka Connect configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-connect` for reference).
* `kafka_rest_settings` - Kafka REST configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-rest` for reference).
* `kafka_settings` - Kafka configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka` for reference).
* `schema_registry_settings` - Schema Registry configuration settings in JSON format (`exo dbaas type show kafka --settings=schema-registry` for reference)
* `version` - Kafka major version (`exo dbaas type show kafka` for reference). Can only be set during creation.

The `mysql` block supports:

* `admin_password` - A custom administrator account password. Can only be set during creation.
* `admin_username` - A custom administrator account username. Can only be set during creation.
* `backup_schedule` - The automated backup schedule (format: HH:MM).
* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `mysql_settings` - MySQL configuration settings in JSON format (`exo dbaas type show mysql --settings=mysql` for reference).
* `version` - MySQL major version (`exo dbaas type show mysql` for reference). Can only be set during creation.

The `pg` block supports:

* `admin_password` - A custom administrator account password. Can only be set during creation.
* `admin_username` - A custom administrator account username. Can only be set during creation.
* `backup_schedule` - The automated backup schedule (format: HH:MM).
* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `pgbouncer_settings` - PgBouncer configuration settings in JSON format (`exo dbaas type show pg --settings=pgbouncer` for reference).
* `pglookout_settings` - pglookout configuration settings in JSON format (`exo dbaas type show pg --settings=pglookout` for reference).
* `pg_settings` - PostgreSQL configuration settings in JSON format (`exo dbaas type show pg --settings=pg` for reference).
* `version` - PostgreSQL major version (`exo dbaas type show pg` for reference). Can only be set during creation.

The `redis` block supports:

* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `redis_settings` - Redis configuration settings in JSON format (`exo dbaas type show redis --settings=redis` for reference).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `created_at` - The creation date of the database service.
* `disk_size` - The disk size of the database service.
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

