data "exoscale_instance_pool_list" "my_instance_pool_list" {
  zone = "ch-gva-2"
}

output "my_instance_pool_ids" {
  value = join("\n", formatlist(
    "%s", exoscale_instance_pool_list.my_instance_pool_list.pools.*.id
  ))
}
