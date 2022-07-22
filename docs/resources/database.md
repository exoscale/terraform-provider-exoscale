---
page_title: "Exoscale: exoscale_database"
description: |-
  Manage Exoscale Database Services (DBaaS).
---

# exoscale\_database

Manage Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).


## Usage

```hcl
resource "exoscale_database" "my_database" {
  zone = "ch-gva-2"
  name = "my-database"

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

output "my_database_uri" {
  value = exoscale_database.my_database.uri
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/
[cli]: https://github.com/exoscale/cli/

* `zone` - (Required) The name of the [zone][zone] to create the database service into.
* `name` - (Required) The name of the database service.
* `type` - (Required) The type of the database service (`kafka`, `mysql`, `opensearch`, `pg`, `redis`).
* `plan` - (Required) The plan of the database service (use the [Exoscale CLI][cli] - `exo dbaas type show <TYPE>` - for reference).

* `maintenance_dow` - The day of week to perform the automated database service maintenance (`never`, `monday`, `tuesday`, `wednesday`, `thursday`, `friday`, `saturday`, `sunday`).
* `maintenance_time` - The time of day to perform the automated database service maintenance (`HH:MM:SS`)
* `termination_protection` - The database service protection boolean flag against termination/power-off.

* `kafka` - (Block) *kafka* database service type specific arguments. Structure is documented below.
* `mysql` - (Block) *mysql* database service type specific arguments. Structure is documented below.
* `opensearch` - (Block) *opensearch* database service type specific arguments. Structure is documented below.
* `pg` - (Block) *pg* database service type specific arguments. Structure is documented below.
* `redis` - (Block) *redis* database service type specific arguments. Structure is documented below.

### `kafka` block

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
* `version` - Kafka major version (`exo dbaas type show kafka` for reference; may only be set at creation time).

### `mysql` block

* `admin_password` - A custom administrator account password (may only be set at creation time).
* `admin_username` - A custom administrator account username (may only be set at creation time).
* `backup_schedule` - The automated backup schedule (`HH:MM`).
* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `mysql_settings` - MySQL configuration settings in JSON format (`exo dbaas type show mysql --settings=mysql` for reference).
* `version` - MySQL major version (`exo dbaas type show mysql` for reference; may only be set at creation time).

### `opensearch` block

* `fork_from_service` -  Service name
* `recovery_backup_name` -
* `index_pattern` -  (can be used multiple times) Allows you to create glob style patterns and set a max number of indexes matching this pattern you want to keep. Creating indexes exceeding this value will cause the oldest one to get deleted. You could for example create a pattern looking like 'logs.?' and then create index logs.1, logs.2 etc, it will delete logs.1 once you create logs.6. Do note 'logs.?' does not apply to logs.10. Note: Setting max_index_count to 0 will do nothing and the pattern gets ignored.
	* `max_index_count` -  Maximum number of indexes to keep (Minimum value is `0`)
	* `pattern` -  fnmatch pattern
	* `sorting_algorithm` - `alphabetical` or `creation_date`.
* `index_template` - Template settings for all new indexes
	* `mapping_nested_objects_limit` -  The maximum number of nested JSON objects that a single document can contain across all nested types. This limit helps to prevent out of memory errors when a document contains too many nested objects. (Default is 10000. Minimum value is `0`, maximum value is `100000`.)
	* `number_of_replicas` -  The number of replicas each primary shard has. (Minimum value is `0`, maximum value is `29`)
	* `number_of_shards` -  The number of primary shards that an index should have. (Minimum value is `1`, maximum value is `1024`.)
* `ip_filter` -  Allow incoming connections from this list of CIDR address block, e.g. `["10.20.0.0/16"]`
* `keep_index_refresh_interval` -  Aiven automation resets index.refresh_interval to default value for every index to be sure that indices are always visible to search. If it doesn't fit your case, you can disable this by setting up this flag to true.
* `max_index_count` -  Maximum number of indexes to keep before deleting the oldest one (Minimum value is `0`)
* `dashboards`
	* `enabled` -                   {Type -  schema.TypeBool, Optional -  true, Default -  true},
	* `max_old_space_size` -           {Type -  schema.TypeInt, Optional -  true, Default -  128},
	* `request_timeout` -  {Type -  schema.TypeInt, Optional -  true, Default -  30000},
`settings` -  OpenSearch-specific settings, in json. e.g.`jsonencode({thread_pool_search_size: 64})`. Use `exo x get-dbaas-settings-opensearch` to get a list of available settings.
* `version` -  OpenSearch major version.

### `pg` block

* `admin_password` - A custom administrator account password (may only be set at creation time).
* `admin_username` - A custom administrator account username (may only be set at creation time).
* `backup_schedule` - The automated backup schedule (`HH:MM`).
* `ip_filter` - A list of CIDR blocks to allow incoming connections from.
* `pgbouncer_settings` - PgBouncer configuration settings in JSON format (`exo dbaas type show pg --settings=pgbouncer` for reference).
* `pglookout_settings` - pglookout configuration settings in JSON format (`exo dbaas type show pg --settings=pglookout` for reference).
* `pg_settings` - PostgreSQL configuration settings in JSON format (`exo dbaas type show pg --settings=pg` for reference).
* `version` - PostgreSQL major version (`exo dbaas type show pg` for reference; may only be set at creation time).

### `redis` block

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
* `updated_at` - The date of the latest database service update.
* `uri` - The database service connection URI.


## Import

An existing database service may be imported by `<name>@<zone>`:

```console
$ terraform import \
  exoscale_database.my_database \
  my-database@ch-gva-2
```
