provider "template" {
  version = "~> 2.1"
}

provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}
