# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone      = "ch-gva-2"
  my_template  = "Linux Ubuntu 22.04 LTS 64-bit"
  my_instances = ["my-instance-1", "my-instance-2"]
}

# Existing resources (<-> data sources)
data "exoscale_compute_template" "my_template" {
  zone = local.my_zone
  name = local.my_template
}

data "exoscale_security_group" "default" {
  name = "default"
}

# cloud-init
data "external" "terraform_version" {
  program = [
    "sh",
    "-c",
    "terraform -v | sed -nE 's|^.*\\sv(.*)$|{\"terraform_version\": \"\\1\"}|p;q'"
  ]
}

data "cloudinit_config" "my_cloud_init" {
  count = length(local.my_instances)

  gzip          = false
  base64_encode = false

  # cloud-config
  part {
    filename     = "init.cfg"
    content_type = "text/cloud-config"
    content = templatefile(
      "cloud-init.yaml.tpl",
      {
        fqdn = "${local.my_instances[count.index]}.example.exo"
      }
    )
  }

  # x-shellscript
  part {
    content_type = "text/x-shellscript"
    content = templatefile(
      "x-shellscript.sh.tpl",
      {
        terraform_version = data.external.terraform_version.result.terraform_version
      }
    )
  }
}

# Sample instance
resource "exoscale_compute_instance" "my_instance" {
  count = length(local.my_instances)

  zone = local.my_zone
  name = local.my_instances[count.index]

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.small"
  disk_size   = 10

  ssh_key   = exoscale_ssh_key.my_ssh_key.name
  user_data = data.cloudinit_config.my_cloud_init[count.index].rendered

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
      "sleep 10", # give cloud-init time
      "grep -Fw Terraform /var/log/cloud-init-output.log",
    ]
  }
}

# Outputs
output "ssh_connection" {
  value = join("\n", formatlist(
    "ssh -i id_ssh %s@%s  # %s",
    data.exoscale_compute_template.my_template.username,
    exoscale_compute_instance.my_instance.*.public_ip_address,
    exoscale_compute_instance.my_instance.*.name,
  ))
}
