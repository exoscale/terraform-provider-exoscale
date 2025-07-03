# Customizable parameters
locals {
  my_zone = "ch-gva-2"
}

resource "exoscale_dbaas" "postgres" {
  name = "databasename"
  plan = "hobbyist-2"
  type = "pg"
  zone = local.my_zone
  pg {
    admin_username = "${var.database_username}"
    admin_password = "${var.database_password}"
    ip_filter = ["0.0.0.0/0"]
    version = "16"
  }
}
