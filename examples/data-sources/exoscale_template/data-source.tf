data "exoscale_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

output "my_template_id" {
  value = data.exoscale_template.my_template.id
}
