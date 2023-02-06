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
resource "exoscale_compute_instance" "my_instance" {
  zone = local.my_zone
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.micro"
  disk_size   = 10

  ssh_key = exoscale_ssh_key.my_ssh_key.name

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]
}

# Outputs
output "ssh_connection" {
  value = format("ssh -i id_ssh %s@%s  # %s",
    data.exoscale_compute_template.my_template.username,
    exoscale_compute_instance.my_instance.public_ip_address,
    exoscale_compute_instance.my_instance.name,
  )
}
