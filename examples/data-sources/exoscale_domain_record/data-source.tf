data "exoscale_domain" "my_domain" {
  name = "my.domain"
}

data "exoscale_domain_record" "my_exoscale_domain_A_records" {
  domain = data.exoscale_domain.my_domain.name
  filter {
    name        = "my-host"
    record_type = "A"
  }
}

data "exoscale_domain_record" "my_exoscale_domain_NS_records" {
  domain = data.exoscale_domain.my_domain.name
  filter {
    content_regex = "ns.*"
  }
}

output "my_exoscale_domain_A_records" {
  value = join("\n", formatlist(
    "%s", data.exoscale_domain_record.my_exoscale_domain_A_records.records.*.name
  ))
}

output "my_exoscale_domain_NS_records" {
  value = join("\n", formatlist(
    "%s", data.exoscale_domain_record.my_exoscale_domain_NS_records.records.*.content
  ))
}
