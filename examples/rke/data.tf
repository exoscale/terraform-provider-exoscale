data "template_file" "init" {
  template = "${file("init.tpl")}"
  count = "${length(var.hostnames)}"

  vars {
    ubuntu = "xenial"
    docker-version = "17.03.2~ce-0~ubuntu-xenial"
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
