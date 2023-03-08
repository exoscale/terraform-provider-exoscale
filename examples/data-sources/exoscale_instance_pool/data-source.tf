data "exoscale_instance_pool" "my_instance_pool" {
  zone = "ch-gva-2"
  name = "my-instance-pool"
}

output "my_instance_pool_id" {
  value = data.exoscale_instance_pool.my_instance_pool.id
}
