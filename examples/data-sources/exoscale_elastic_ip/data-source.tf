data "exoscale_elastic_ip" "my_elastic_ip" {
  zone       = "ch-gva-2"
  ip_address = "1.2.3.4"
}

output "my_elastic_ip_id" {
  value = data.exoscale_elastic_ip.my_elastic_ip.id
}
