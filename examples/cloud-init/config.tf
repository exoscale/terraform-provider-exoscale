provider "template" {
  version = "~> 1.0"
}

provider "exoscale" {
  version = "~> 0.9.41"
  key = "${var.key}"
  secret = "${var.secret}"
}
