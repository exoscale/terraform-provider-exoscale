provider "template" {
  version = "~> 2.1"
}

provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}

data "exoscale_compute_template" "exokube" {
  zone = var.zone
  name = var.template
}

resource "exoscale_compute" "exokube" {
  display_name = "exokube"
  size = "Medium"
  disk_size = 50
  zone = var.zone
  template_id = data.exoscale_compute_template.exokube.id
  key_pair = var.key_pair
  ip6 = true

  security_groups = [
    exoscale_security_group.exokube.name,
  ]

  user_data = data.template_cloudinit_config.exokube.rendered
}
