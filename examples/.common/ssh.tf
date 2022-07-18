# SSH key
resource "tls_private_key" "my_ssh_key" {
  algorithm = "ED25519"
}

resource "exoscale_ssh_key" "my_ssh_key" {
  name       = "my-ssh-key"
  public_key = tls_private_key.my_ssh_key.public_key_openssh
}

resource "local_sensitive_file" "my_ssh_private_key" {
  filename        = "id_ssh"
  content         = tls_private_key.my_ssh_key.private_key_openssh
  file_permission = "0600"
}

# SSH security group
resource "exoscale_security_group" "my_ssh_security_group" {
  name = "my-ssh-security-group"
}

resource "exoscale_security_group_rule" "ssh_ipv4" {
  security_group_id = exoscale_security_group.my_ssh_security_group.id
  description       = "SSH (IPv4)"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "ssh_ipv6" {
  security_group_id = exoscale_security_group.my_ssh_security_group.id
  description       = "SSH (IPv6)"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 22
  end_port          = 22
  cidr              = "::/0"
}
