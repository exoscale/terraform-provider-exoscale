data "exoscale_private_network" "my_private_network" {
  zone = "ch-gva-2"
  name = "my-private-network"
}

output "my_private_network_id" {
  value = data.exoscale_private_network.my_private_network.id
}
