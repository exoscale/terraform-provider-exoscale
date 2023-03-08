data "exoscale_domain" "my_domain" {
  name = "my.domain"
}

output "my_domain_id" {
  value = data.exoscale_domain.my_domain.id
}
