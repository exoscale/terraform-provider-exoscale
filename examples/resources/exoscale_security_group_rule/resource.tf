resource "exoscale_security_group" "my_security_group" {
  name = "my-security-group"
}

resource "exoscale_security_group_rule" "my_security_group_rule" {
  security_group_id = exoscale_security_group.my_security_group.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0" # "::/0" for IPv6
  start_port        = 80
  end_port          = 80
}
