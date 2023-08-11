# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone = "ch-gva-2"
}

# Read local IP address
data "http" "myip" {
  url = "https://ifconfig.me/ip"
}

resource "exoscale_database" "my_database" {
  zone = local.my_zone
  name = "my-database"

  type = "grafana"
  plan = "hobbyist-2"

  termination_protection = false

  grafana {
    ip_filter = [
      "${data.http.myip.response_body}/32"
    ]
  }
}

data "exoscale_database_uri" "my_database" {
  zone = local.my_zone
  name = exoscale_database.my_database.name
  type = "grafana"
}


# Outputs
output "database_uri" {
  value     = data.exoscale_database_uri.my_database.uri
  sensitive = true
}
