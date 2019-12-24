provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}

data "exoscale_compute_template" "main" {
  zone = var.zone
  name = var.template
}

resource "exoscale_security_group" "default" {
  name = "default-with-ipv6"
}

resource "exoscale_security_group_rule" "default-ssh-4" {
  description = "ssh -4"
  security_group_id = exoscale_security_group.default.id
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 22
  end_port = 22
}

resource "exoscale_security_group_rule" "default-ssh-6" {
  description = "ssh -6"
  security_group_id = exoscale_security_group.default.id
  protocol = "TCP"
  type = "INGRESS"
  cidr = "::/0"
  start_port = 22
  end_port = 22
}

resource "exoscale_compute" "main" {
  display_name = "test-ipv6"
  template_id = data.exoscale_compute_template.main.id
  zone = var.zone
  size = "Medium"
  disk_size = 11

  ip6 = true

  key_pair = var.key_pair
  user_data = <<EOF
#cloud-config
manage_etc_hosts: localhost
EOF

  security_groups = [exoscale_security_group.default.name]
}

output "username" {
  value = exoscale_compute.main.username
}

output "ip_address" {
  value = exoscale_compute.main.ip_address
}

output "ip6_address" {
  value = exoscale_compute.main.ip6_address
}
