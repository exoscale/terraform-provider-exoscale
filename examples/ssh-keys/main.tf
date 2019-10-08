provider "exoscale" {
  version = "~> 0.11"
}

resource "exoscale_ssh_keypair" "key" {
  name = "mykey"
}

data "exoscale_compute_template" "ubuntu" {
  zone = "at-vie-1"
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "vm" {
  display_name = "myvm"
  template_id = "${data.exoscale_compute_template.ubuntu.id}"
  size = "Medium"
  key_pair = "${exoscale_ssh_keypair.key.name}"
  disk_size = 10
  zone = "at-vie-1"

  provisioner "remote-exec" {
    connection {
      private_key = "${exoscale_ssh_keypair.key.private_key}"
    }

    inline = [
      "uname -a"
    ]
  }
}
