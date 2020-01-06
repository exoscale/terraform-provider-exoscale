variable "machines" {
  default = 2
}

data "exoscale_compute_template" "ubuntu" {
  zone = var.zone
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "machine" {
  count = var.machines

  display_name = "demo-machine-${count.index}"

  template_id = data.exoscale_compute_template.ubuntu.id
  size = "Small"
  disk_size = "10"
  security_groups = ["default"]
  key_pair = var.key_pair
  zone = var.zone

  user_data = element(data.template_file.user_data.*.rendered, count.index)
}

resource "exoscale_nic" "eth_intra" {
  count = length(exoscale_compute.machine)

  compute_id = exoscale_compute.machine.*.id[count.index]
  network_id = exoscale_network.intra.id
}
