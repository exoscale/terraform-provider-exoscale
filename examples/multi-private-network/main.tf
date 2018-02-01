provider "template" {
  version = "~> 1.0"
}

provider "exoscale" {
  version = "~> 0.9.8"
  token = "${var.token}"
  secret = "${var.secret}"
}
