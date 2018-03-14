resource "exoscale_compute" "ada" {
  display_name = "ada-lovelace"
  key_pair = "my@keypair"
  disk_size = 10
  size = "Tiny"
  template = "Linux Debian 9 64-bit"
  zone = "ch-dk-2"

  timeouts {
    create = "30s"
    delete = "2h"
  }

  security_groups = ["${exoscale_security_group.default.name}"]
}
