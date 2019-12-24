resource "exoscale_security_group" "default" {
  name = "default"
  description = "Default Security Group"
}

resource "exoscale_security_group_rule" "default" {
  type = "INGRESS"
  security_group_id = exoscale_security_group.default.id
  protocol = "ICMP"
  icmp_type = 8
  icmp_code = 0
  cidr = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "default-1" {
  type = "INGRESS"
  security_group_id = exoscale_security_group.default.id
  protocol = "TCP"
  start_port = 22
  end_port = 22
  cidr = "0.0.0.0/0"
}
