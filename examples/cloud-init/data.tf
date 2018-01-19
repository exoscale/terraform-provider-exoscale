data "template_file" "init" {
  template = "${file("init.tpl")}"
  count = "${var.master}"

  vars {
    ubuntu = "artful"
    fqdn = "${element(var.hostnames, count.index)}"
  }
}

data "template_cloudinit_config" "config" {
  gzip = true
  base64_encode = true

  count = "${var.master}"

  part {
    filename = "init.cfg"
    content_type = "text/cloud-config"
    content = "${element(data.template_file.init.*.rendered, count.index)}"
  }
}
