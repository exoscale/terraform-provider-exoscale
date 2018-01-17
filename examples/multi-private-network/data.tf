data "template_file" "user_data" {
  template = "${file("cloud-config.tpl")}"
  count = "${var.machines}"

  vars {
    hostname = "demo-machine-${count.index}"
    ip_address = "192.168.0.${format("%d", 1 + count.index)}"
    netmask = "255.255.255.0"
    gateway = "192.168.0.255"
  }
}
