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

resource "exoscale_network" "intra" {
  name = "demo-intra"
  display_text = "demo intra privnet"
  zone = "${var.zone}"
  network_offering = "PrivNet"

  start_ip = "10.0.0.50"
  end_ip = "10.0.0.250"
  netmask = "255.255.255.0"
}

resource "exoscale_compute" "static" {
  count = "${var.static_machines}"

  display_name = "demo-static-${count.index}"

  template = "Linux Ubuntu 18.04 LTS 64-bit"
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = "${var.key_pair}"
  zone = "${var.zone}"

  user_data = "${file("cloud-config.yaml")}"
}

resource "exoscale_nic" "eth_static" {
  count = "${exoscale_compute.static.count}"

  compute_id = "${exoscale_compute.static.*.id[count.index]}"
  network_id = "${exoscale_network.intra.id}"

  # static IP address
  ip_address = "${format("10.0.0.%d", count.index + 1)}"
}

resource "exoscale_compute" "dynamic" {
  count = "${var.dynamic_machines}"

  display_name = "demo-dynamic-${count.index}"

  template = "Linux Ubuntu 18.04 LTS 64-bit"
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = "${var.key_pair}"
  zone = "${var.zone}"

  user_data = "${file("cloud-config.yaml")}"
}

resource "exoscale_nic" "eth_dynamic" {
  count = "${exoscale_compute.dynamic.count}"

  compute_id = "${exoscale_compute.dynamic.*.id[count.index]}"
  network_id = "${exoscale_network.intra.id}"
}

provider "exoscale" {
  version = "~> 0.9.36"
  key = "${var.key}"
  secret = "${var.secret}"
}
