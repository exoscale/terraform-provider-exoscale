resource "exoscale_dbaas" "my_kafka" {
  zone = "ch-gva-2"
  name = "my-kafka"
  type = "kafka"
  plan = "startup-4"
}

resource "exoscale_dbaas_kafka_user" "my_app" {
  zone     = exoscale_dbaas.my_kafka.zone
  service  = exoscale_dbaas.my_kafka.name
  username = "my-app"
}

resource "exoscale_dbaas_kafka_topic_acl" "my_app_orders" {
  zone       = exoscale_dbaas.my_kafka.zone
  service    = exoscale_dbaas.my_kafka.name
  username   = exoscale_dbaas_kafka_user.my_app.username
  topic      = "orders-*"
  permission = "readwrite"
}
