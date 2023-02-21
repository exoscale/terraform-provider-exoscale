resource "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name        = "my-anti-affinity-group"
  description = "Prevent compute instances to run on the same host"
}
