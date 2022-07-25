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

# SSH
# -> ssh.tf

# Sample instance (mark: "ipv6 = true")
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10
  ipv6        = true

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

    inline = [
      "sleep 3", # give IPv6 SLAAC some time
      "ip address show dev eth0",
    ]
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

output "my_instance_ipv4" {
  value = exoscale_compute_instance.my_instance.public_ip_address
}

output "my_instance_ipv6" {
  value = exoscale_compute_instance.my_instance.ipv6_address
}
