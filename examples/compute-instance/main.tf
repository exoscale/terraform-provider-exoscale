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

# Sample instance pool
resource "exoscale_compute_instance" "my_big_instance" {
  zone = local.my_zone
  name = "my-big-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  ssh_key = exoscale_ssh_key.my_ssh_key.name

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]
}

resource "exoscale_compute_instance" "my_small_instance" {
  zone = local.my_zone
  name = "my-small-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.micro"
  disk_size   = 10

  ssh_key = exoscale_ssh_key.my_ssh_key.name

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]
}

data "exoscale_compute_instance_list" "small_instances" {
  zone = "ch-gva-2"
  filter {
    name  = "name"
    value = "my-small-instance"
  }
}

# Outputs
output "ssh_connection" {
  value = join("\n",
    formatlist("ssh -i id_ssh %s@%s  # %s",
      data.exoscale_compute_template.my_template.username,
      data.exoscale_compute_instance_list.small_instances.instances.*.public_ip_address,
      data.exoscale_compute_instance_list.small_instances.instances.*.name,
    )
  )
}
