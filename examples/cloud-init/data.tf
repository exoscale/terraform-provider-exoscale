data "template_file" "init" {
  template = "${file("init.tpl")}"
  count = "${length(var.hostnames)}"

  vars {
    ubuntu = "artful"
    fqdn = "${element(var.hostnames, count.index)}"
  }
}

data "template_cloudinit_config" "config" {
  gzip = true
  base64_encode = true

  count = "${length(var.hostnames)}"

  part {
    filename = "init.cfg"
    content_type = "text/cloud-config"
    content = "${element(data.template_file.init.*.rendered, count.index)}"
  }
}
