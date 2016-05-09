provider "exoscale" {
  token = "EXOc511a0b8c4d69ea14cbae879"
  secret = "STBlGbZEOBN__nBXB36vPNBG3Y6eDTqyUJ26MavisWo"
}

resource "exoscale_ssh" {
    name = "terraform-key"
}

resource "exoscale_compute" "test" {
    template = "ubuntu-16-04-x64"
    name = "test-1"
    zone = "CH-DK-2"
    size = "Micro"
    keypair = "kusanagi"
}
