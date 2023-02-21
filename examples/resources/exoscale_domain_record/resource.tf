resource "exoscale_domain" "my_domain" {
  name = "example.net"
}

resource "exoscale_domain_record" "my_host" {
  domain      = exoscale_domain.my_domain.id
  name        = "my-host"
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "my_host_alias" {
  domain      = exoscale_domain.my_domain.id
  name        = "my-host-alias"
  record_type = "CNAME"
  content     = exoscale_domain_record.my_host.hostname
}
