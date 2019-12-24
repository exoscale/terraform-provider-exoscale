data "exoscale_compute_template" "test" {
  zone = var.zone
  name = var.template
}

resource "exoscale_compute" "test" {
  count = length(var.hostnames)
  display_name = var.hostnames[count.index]
  template_id = data.exoscale_compute_template.test.id
  zone = var.zone
  size = "Tiny"
  disk_size = 17

  key_pair = var.key_pair
  security_groups = ["default"]

  user_data = element(data.template_cloudinit_config.config.*.rendered, count.index)
}
