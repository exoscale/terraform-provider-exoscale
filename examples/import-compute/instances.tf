data "exoscale_compute_template" "debian" {
  zone = "ch-dk-2"
  name = "Linux Debian 9 64-bit"
}

resource "exoscale_compute" "ada" {
  display_name = "ada-lovelace"
  key_pair = "my@keypair"
  disk_size = 10
  size = "Tiny"
  template_id = data.exoscale_compute_template.debian.id
  zone = "ch-dk-2"

  timeouts {
    create = "30s"
    delete = "2h"
  }

  security_groups = [exoscale_security_group.default.name]
}
