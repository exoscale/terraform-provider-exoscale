data "exoscale_compute_instance_list" "my_compute_instance_list" {
  zone = "ch-gva-2"

  type = "standard.micro"

  name = "/.*ubuntu.*/"

  labels = {
    "customer" = "/.*bank.*/"
    "contract" = "premium-support"
  }
}

output "my_compute_instance_ids" {
  value = join("\n", formatlist(
    "%s", data.exoscale_compute_instance_list.my_compute_instance_list.instances.*.id
  ))
}
