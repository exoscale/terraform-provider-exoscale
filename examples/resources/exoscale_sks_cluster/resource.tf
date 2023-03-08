resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = "ch-gva-2"
  name = "my-sks-cluster"
}

output "my_sks_cluster_endpoint" {
  value = exoscale_sks_cluster.my_sks_cluster.endpoint
}
