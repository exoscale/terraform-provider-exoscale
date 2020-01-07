provider "template" {
  version = "~> 2.1"
}

provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}

data "exoscale_compute_template" "ubuntu" {
  zone = var.zone
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_ipaddress" "ingress" {
  zone = var.zone
  description = "my elastic IP"
}

data "template_file" "cloudinit" {
  template = file("init.tpl")

  vars = {
    eip = exoscale_ipaddress.ingress.ip_address
  }
}

resource "exoscale_compute" "machine" {
  display_name = "machine"
  template_id = data.exoscale_compute_template.ubuntu.id
  size = "Medium"
  disk_size = "22"
  key_pair = var.key_pair
  zone = var.zone

  security_groups = ["default"]
  user_data = data.template_file.cloudinit.rendered
}

resource "exoscale_secondary_ipaddress" "machine" {
  compute_id = exoscale_compute.machine.id
  ip_address = exoscale_ipaddress.ingress.ip_address
}

output "connection" {
  value = format("%s@%s", exoscale_compute.machine.username, exoscale_compute.machine.ip_address)
}
