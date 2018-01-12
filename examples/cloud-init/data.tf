data "template_file" "cloud_init" {
  template = "${file("init.tpl")}"

  vars {
    ubuntu = "artful"
  }
}
