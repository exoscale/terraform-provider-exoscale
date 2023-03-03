# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone     = "ch-gva-2"
  my_template = "Linux Ubuntu 22.04 LTS 64-bit"
}

# Existing resources (<-> data sources)
data "exoscale_template" "my_template" {
  zone = local.my_zone
  name = local.my_template
}

data "exoscale_security_group" "default" {
  name = "default"
}

# Sample instance
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10
  ipv6        = true

  security_group_ids = [
    data.exoscale_security_group.default.id,
  ]
}

# DNS
resource "exoscale_domain" "my_domain" {
  name = "example.exo"
}

resource "exoscale_domain_record" "root_ipv4" {
  domain      = exoscale_domain.my_domain.id
  name        = ""
  record_type = "A"
  content     = exoscale_compute_instance.my_instance.public_ip_address
}

resource "exoscale_domain_record" "root_ipv6" {
  domain      = exoscale_domain.my_domain.id
  name        = ""
  record_type = "AAAA"
  content     = exoscale_compute_instance.my_instance.ipv6_address
}

resource "exoscale_domain_record" "www" {
  domain      = exoscale_domain.my_domain.id
  name        = "www"
  record_type = "CNAME"
  content     = exoscale_domain.my_domain.name
  ttl         = 7200
}

resource "exoscale_domain_record" "hello" {
  domain      = exoscale_domain.my_domain.id
  name        = ""
  record_type = "TXT"
  content     = "hello world!"
}

# Outputs
output "my_instance_ipv4" {
  value = exoscale_compute_instance.my_instance.public_ip_address
}

output "my_instance_ipv6" {
  value = exoscale_compute_instance.my_instance.ipv6_address
}
