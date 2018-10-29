data "template_file" "user_data" {
  template = "${file("cloud-config.yaml")}"
  count = "${var.machines}"

  vars {
    hostname = "demo-machine-${count.index}"
    ip_address = "192.168.0.${format("%d", 1 + count.index)}/24"
  }
}
