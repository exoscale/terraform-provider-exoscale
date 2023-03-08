data "exoscale_compute_instance" "my_instance" {
  zone = "ch-gva-2"
  name = "my-instance"
}

output "my_instance_id" {
  value = data.exoscale_compute_instance.my_instance.id
}
