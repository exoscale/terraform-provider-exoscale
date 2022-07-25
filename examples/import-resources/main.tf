# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone     = "ch-gva-2"
  my_template = "Linux Ubuntu 22.04 LTS 64-bit"
}

# Existing resources (<-> data sources)
data "exoscale_compute_template" "my_template" {
  zone = local.my_zone
  name = local.my_template
}

# Existing security group ("default") to import
resource "exoscale_security_group" "default" {
  name        = "default"
  description = "Default Security Group"
}

resource "exoscale_security_group_rule" "default" {
  security_group_id = exoscale_security_group.default.id
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "default-1" {
  security_group_id = exoscale_security_group.default.id
  type              = "INGRESS"
  protocol          = "ICMP"
  icmp_type         = 8
  icmp_code         = 0
  cidr              = "0.0.0.0/0"
}

# Existing instance to import
# (please create it "manually" beforehands)
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  security_group_ids = [
    exoscale_security_group.default.id,
  ]
}
