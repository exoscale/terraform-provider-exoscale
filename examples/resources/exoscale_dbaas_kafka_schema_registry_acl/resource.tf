resource "exoscale_dbaas" "my_kafka" {
  zone = "ch-gva-2"
  name = "my-kafka"
  type = "kafka"
  plan = "startup-4"

  kafka = {
    enable_schema_registry = true
  }
}

resource "exoscale_dbaas_kafka_user" "my_app" {
  zone     = exoscale_dbaas.my_kafka.zone
  service  = exoscale_dbaas.my_kafka.name
  username = "my-app"
}

resource "exoscale_dbaas_kafka_schema_registry_acl" "my_app_orders" {
  zone       = exoscale_dbaas.my_kafka.zone
  service    = exoscale_dbaas.my_kafka.name
  username   = exoscale_dbaas_kafka_user.my_app.username
  resource   = "Subject:orders-*"
  permission = "schema_registry_read"
}
