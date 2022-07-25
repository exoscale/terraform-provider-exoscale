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

# Elastic IP
resource "exoscale_elastic_ip" "my_elastic_ip" {
  zone = local.my_zone
}

# Network configuration (<-> cloud-init)
data "cloudinit_config" "my_cloud_config" {
  gzip          = false
  base64_encode = false

  # cloud-config
  part {
    filename     = "init.cfg"
    content_type = "text/cloud-config"
    content = templatefile(
      "cloud-config.yaml.tpl",
      {
        eip = exoscale_elastic_ip.my_elastic_ip.ip_address
      }
    )
  }
}

# SSH
# -> ssh.tf

# Sample instance
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  ssh_key   = exoscale_ssh_key.my_ssh_key.name
  user_data = data.cloudinit_config.my_cloud_config.rendered

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]

  elastic_ip_ids = [exoscale_elastic_ip.my_elastic_ip.id]

  provisioner "remote-exec" {
    connection {
      host        = self.public_ip_address
      user        = data.exoscale_compute_template.my_template.username
      private_key = tls_private_key.my_ssh_key.private_key_openssh
    }

    inline = [
      "sleep 10", # give cloud-init time
      "ip -4 address show dev lo",
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

output "my_elastic_ip" {
  value = exoscale_elastic_ip.my_elastic_ip.ip_address
}
