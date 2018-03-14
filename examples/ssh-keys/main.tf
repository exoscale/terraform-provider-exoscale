provider "exoscale" {
  version = "~> 0.9.14"
}

resource "exoscale_ssh_keypair" "key" {
  name = "mykey"
}

resource "exoscale_compute" "vm" {
  display_name = "myvm"
  template = "Linux Ubuntu 17.10 64-bit"
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
