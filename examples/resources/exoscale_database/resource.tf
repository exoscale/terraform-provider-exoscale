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
      timezone : "Europe/Zurich"
    })
  }
}

output "my_database_uri" {
  value = exoscale_database.my_database.uri
}
