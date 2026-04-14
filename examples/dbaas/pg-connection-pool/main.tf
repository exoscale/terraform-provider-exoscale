terraform {
  required_providers {
    exoscale = {
      source = "exoscale/exoscale"
    }
  }
}

variable "zone" {
  type    = string
  default = "ch-gva-2"
}

variable "plan" {
  type    = string
  default = "hobbyist-2"
}

variable "pg_version" {
  type    = string
  default = "16"
}

variable "allowed_cidr" {
  type    = string
  default = "0.0.0.0/0"
}

provider "exoscale" {}

locals {
  suffix        = substr(md5(path.cwd), 0, 8)
  service_name  = "example-pg-pool-${local.suffix}"
  database_name = "example_pg_${local.suffix}"
  username      = "example_pg_${local.suffix}"
  pool_name     = "example-pg-pool-${local.suffix}"
}

resource "exoscale_dbaas" "pg" {
  name                   = local.service_name
  plan                   = var.plan
  type                   = "pg"
  zone                   = var.zone
  termination_protection = false

  pg = {
    ip_filter = [var.allowed_cidr]
    version   = var.pg_version

    pgbouncer_settings = jsonencode({
      autodb_pool_mode          = "transaction"
      max_prepared_statements   = 29
      min_pool_size             = 10
      server_idle_timeout       = 500
      server_lifetime           = 3555
      server_reset_query_always = false
    })
  }
}

resource "exoscale_dbaas_pg_database" "app" {
  database_name = local.database_name
  service       = exoscale_dbaas.pg.name
  zone          = exoscale_dbaas.pg.zone
}

resource "exoscale_dbaas_pg_user" "app" {
  username = local.username
  service  = exoscale_dbaas.pg.name
  zone     = exoscale_dbaas.pg.zone
}

resource "exoscale_dbaas_pg_connection_pool" "app" {
  name          = local.pool_name
  database_name = exoscale_dbaas_pg_database.app.database_name
  service       = exoscale_dbaas.pg.name
  zone          = exoscale_dbaas.pg.zone
  username      = exoscale_dbaas_pg_user.app.username
  mode          = "session"
  size          = 10
}

output "service_name" {
  value = exoscale_dbaas.pg.name
}

output "database_name" {
  value = exoscale_dbaas_pg_database.app.database_name
}

output "pool_name" {
  value = exoscale_dbaas_pg_connection_pool.app.name
}

output "pool_username" {
  value = exoscale_dbaas_pg_connection_pool.app.username
}

output "pool_mode" {
  value = exoscale_dbaas_pg_connection_pool.app.mode
}

output "pool_size" {
  value = exoscale_dbaas_pg_connection_pool.app.size
}

output "pool_connection_uri" {
  value     = exoscale_dbaas_pg_connection_pool.app.connection_uri
  sensitive = true
}
