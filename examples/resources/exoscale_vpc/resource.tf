locals {
  zone = "ch-gva-2"
}

resource "exoscale_vpc" "my_vpc" {
  zone        = local.zone
  name        = "my-vpc"
  description = "My Virtual Private Cloud"

  labels = {
    environment = "production"
  }
}
