variable "key" {}
variable "secret" {}
variable "key_pair" {}

variable "zone" {
  default = "ch-gva-2"
}

variable "static_machines" {
  default = 2
}

variable "dynamic_machines" {
  default = 2
}

data "exoscale_compute_template" "ubuntu" {
  zone = var.zone
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_network" "intra" {
  name = "demo-intra"
  display_text = "demo intra privnet"
  zone = var.zone

  start_ip = "10.0.0.50"
  end_ip = "10.0.0.250"
  netmask = "255.255.255.0"
}

resource "exoscale_compute" "static" {
  count = var.static_machines

  display_name = "demo-static-${count.index}"

  template_id = data.exoscale_compute_template.ubuntu.id
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = var.key_pair
  zone = var.zone

  user_data = file("cloud-config.yaml")
}

resource "exoscale_nic" "eth_static" {
  count = length(exoscale_compute.static)

  compute_id = exoscale_compute.static.*.id[count.index]
  network_id = exoscale_network.intra.id

  # static IP address
  ip_address = format("10.0.0.%d", count.index + 1)
}

resource "exoscale_compute" "dynamic" {
  count = var.dynamic_machines

  display_name = "demo-dynamic-${count.index}"

  template_id = data.exoscale_compute_template.ubuntu.id
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = var.key_pair
  zone = var.zone

  user_data = file("cloud-config.yaml")
}

resource "exoscale_nic" "eth_dynamic" {
  count = length(exoscale_compute.dynamic)

  compute_id = exoscale_compute.dynamic.*.id[count.index]
  network_id = exoscale_network.intra.id
}

provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}
