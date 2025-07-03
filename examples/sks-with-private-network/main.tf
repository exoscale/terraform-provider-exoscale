# Customizable parameters
locals {
  my_zone = "ch-gva-2"
}

resource "exoscale_private_network" "my_private_network" {
  zone = local.my_zone
  name = "test-private-network"

  netmask  = "255.255.252.0"
  start_ip = "172.16.0.20"
  end_ip   = "172.16.3.253"
}

resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = local.my_zone
  name = "my-sks-cluster"
}

resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  cluster_id          = exoscale_sks_cluster.my_sks_cluster.id
  zone                = exoscale_sks_cluster.my_sks_cluster.zone
  name                = "my-sks-nodepool"

  instance_type       = "standard.medium"
  size                = 3
  private_network_ids = [exoscale_private_network.my_private_network.id]
}
