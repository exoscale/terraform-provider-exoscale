variable "machines" {
  default = 2
}

resource "exoscale_compute" "machine" {
  count = "${var.machines}"

  display_name = "demo-machine-${count.index}"

  template = "Linux Ubuntu 18.04 LTS 64-bit"
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = "${var.key_pair}"
  zone = "${var.zone}"

  user_data = "${element(data.template_file.user_data.*.rendered, count.index)}"
}

resource "exoscale_nic" "eth_intra" {
  count = "${exoscale_compute.machine.count}"

  compute_id = "${exoscale_compute.machine.*.id[count.index]}"
  network_id = "${exoscale_network.intra.id}"
}
