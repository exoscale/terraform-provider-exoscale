data "exoscale_security_group" "my_security_group" {
  name = "my-security-group"
}

output "my_security_group_id" {
  value = data.exoscale_security_group.my_security_group.id
}
