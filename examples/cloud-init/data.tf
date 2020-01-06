data "template_file" "init" {
  template = file("init.tpl")
  count = length(var.hostnames)

  vars = {
    ubuntu = var.flavor
    fqdn = element(var.hostnames, count.index)
  }
}

data "template_cloudinit_config" "config" {
  gzip = false
  base64_encode = false

  count = length(var.hostnames)

  part {
    filename = "init.cfg"
    content_type = "text/cloud-config"
    content = element(data.template_file.init.*.rendered, count.index)
  }
}
