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

data "exoscale_security_group" "default" {
  name = "default"
}

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

# Sample instance
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  ssh_key = exoscale_ssh_key.my_ssh_key.name

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]

  provisioner "remote-exec" {
    connection {
      host        = self.public_ip_address
      user        = data.exoscale_compute_template.my_template.username
      private_key = tls_private_key.my_ssh_key.private_key_openssh
    }

    inline = ["uname -a"]
  }
}

# Outputs
output "ssh_connection" {
  value = format(
    "ssh -i id_ssh %s@%s",
    data.exoscale_compute_template.my_template.username,
    exoscale_compute_instance.my_instance.public_ip_address,
  )
}
