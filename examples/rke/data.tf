data "template_file" "init" {
  template = "${file("init.tpl")}"
  count = "${length(var.hostnames)}"

  vars {
    ubuntu = "${var.ubuntu-flavor}"
    docker-version = "${var.docker-version}~ce-0~ubuntu-${var.ubuntu-flavor}"
    hostname = "${element(var.hostnames, count.index)}"
  }
}

data "template_cloudinit_config" "config" {
  count = "${length(var.hostnames)}"

  gzip = false
  base64_encode = false

  part {
    filename = "init.cfg"
    content_type = "text/cloud-config"
    content = "${element(data.template_file.init.*.rendered, count.index)}"
  }
}
