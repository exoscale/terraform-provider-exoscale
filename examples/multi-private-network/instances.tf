variable "machines" {
  default = 2
}

resource "exoscale_network" "intra" {
  name = "demo-intra"
  display_text = "demo intra privnet"
  zone = "${var.zone}"
  network_offering = "PrivNet"
}

resource "exoscale_compute" "machine" {
  count = "${var.machines}"

  display_name = "demo-machine-${count.index}"

  template = "Linux Ubuntu 17.10 64-bit"
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = "${var.keypair}"
  zone = "${var.zone}"
}

resource "exoscale_nic" "" {
  count = "${exoscale_compute.machine.count}"

  compute_id = "${exoscale_compute.machine.*.id[count.index]}"
  network_id = "${exoscale_network.intra.id}"
}
