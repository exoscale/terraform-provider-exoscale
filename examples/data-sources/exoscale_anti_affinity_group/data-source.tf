data "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name = "my-anti-affinity-group"
}

output "my_anti_affinity_group_id" {
  value = data.exoscale_anti_affinity_group.my_anti_affinity_group.id
}
