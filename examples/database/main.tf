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
  name = "some-database"

  type = "valkey"
  plan = "hobbyist-2"

  termination_protection = false

  valkey {
    ip_filter = [
      "${data.http.myip.response_body}/32",
      "2.2.3.5/32"
    ]

    valkey_settings = jsonencode({
      io_threads = 1
      acl_channels_default = "allchannels"
      lfu_decay_time = 5
      lfu_log_factor = 7
      maxmemory_policy = "volatile-ttl"
      notify_keyspace_events = "AKE"
      number_of_databases = 11
      persistence = "off"
      pubsub_client_output_buffer_limit = 150
      ssl = true
      timeout = 250
    })

  }
}

data "exoscale_database_uri" "my_database" {
  zone = local.my_zone
  name = exoscale_database.my_database.name
  type = "valkey"
}


# Outputs
output "database_credentials" {
  value = {
    uri      = data.exoscale_database_uri.my_database.uri
    username = data.exoscale_database_uri.my_database.username
    password = data.exoscale_database_uri.my_database.password
  }
  sensitive = true
}
