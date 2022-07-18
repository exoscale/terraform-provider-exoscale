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

# Managed private network
resource "exoscale_private_network" "my_private_network" {
  zone = local.my_zone
  name = "my-private-network"

  netmask  = "255.255.255.0"
  start_ip = "10.0.0.50"
  end_ip   = "10.0.0.250"
}

# SSH
# -> ssh.tf

# Sample instances
resource "exoscale_compute_instance" "my_instance_static" {
  zone = local.my_zone
  name = "my-instance-static"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.small"
  disk_size   = 10

  ssh_key   = exoscale_ssh_key.my_ssh_key.name
  user_data = file("cloud-config.yaml")

  security_group_ids = [data.exoscale_security_group.default.id]

  network_interface {
    network_id = exoscale_private_network.my_private_network.id
    ip_address = "10.0.0.1"
  }

  provisioner "remote-exec" {
    connection {
      host        = self.public_ip_address
      user        = data.exoscale_compute_template.my_template.username
      private_key = tls_private_key.my_ssh_key.private_key_openssh
    }

    inline = [
      "sleep 10", # give cloud-init and DHCP time
      "ip -4 address show dev eth1",
    ]
  }
}

resource "exoscale_compute_instance" "my_instance_dynamic" {
  zone = local.my_zone
  name = "my-instance-dynamic"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.small"
  disk_size   = 10

  ssh_key   = exoscale_ssh_key.my_ssh_key.name
  user_data = file("cloud-config.yaml")

  security_group_ids = [data.exoscale_security_group.default.id]

  network_interface {
    network_id = exoscale_private_network.my_private_network.id
  }

  provisioner "remote-exec" {
    connection {
      host        = self.public_ip_address
      user        = data.exoscale_compute_template.my_template.username
      private_key = tls_private_key.my_ssh_key.private_key_openssh
    }

    inline = [
      "sleep 10", # give cloud-init and DHCP time
      "ip -4 address show dev eth1",
    ]
  }
}

# Outputs
output "ssh_connection_static" {
  value = format(
    "ssh -i id_ssh %s@%s",
    data.exoscale_compute_template.my_template.username,
    exoscale_compute_instance.my_instance_static.public_ip_address,
  )
}

output "ssh_connection_dynamic" {
  value = format(
    "ssh -i id_ssh %s@%s",
    data.exoscale_compute_template.my_template.username,
    exoscale_compute_instance.my_instance_dynamic.public_ip_address,
  )
}
