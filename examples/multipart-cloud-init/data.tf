data "template_file" "init" {
  count = "${length(var.hostnames)}"
  template = "${file("cloud-init.yml.tpl")}"

  vars {
    fqdn = "${var.hostnames[count.index]}"
    ubuntu = "artful"
  }
}

data "external" "terraform_version" {
  program = [
    "sh",
    "-c",
    "echo \"{\\\"script\\\": \\\"#!/bin/sh\\\\necho Setup via $(terraform -v | head -n 1)\\\"}\""
  ]
}

data "template_cloudinit_config" "config" {
  count = "${length(var.hostnames)}"

  part {
    filename = "cloud-init.yml"
    content_type = "text/cloud-config"
    content = "${element(data.template_file.init.*.rendered, count.index)}"
  }

  part {
    content_type = "text/x-shellscript"
    content = "${data.external.terraform_version.result.script}"
  }
}
