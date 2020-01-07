provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}

resource "exoscale_domain" "exo" {
  name = "example.exo"
}

resource "exoscale_domain_record" "root" {
  domain = exoscale_domain.exo.name
  content = "159.100.200.1"
  name = ""
  record_type = "A"
}

resource "exoscale_domain_record" "www" {
  domain = exoscale_domain.exo.name
  content = exoscale_domain_record.root.hostname
  name = "www"
  record_type = "CNAME"
  ttl = 7200
}

resource "exoscale_domain_record" "hello" {
  domain = exoscale_domain.exo.name
  content = "hello world!"
  name = ""
  record_type = "TXT"
}
