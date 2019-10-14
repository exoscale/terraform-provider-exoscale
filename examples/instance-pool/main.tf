variable "zone" {
  default = "ch-gva-2"
}


resource "exoscale_instance_pool" "instancepool-test" {
  name = "terraforminstancepool"
  template = ""
  serviceoffering = "Medium"
  size = 3
  key_pair = "test"
  zone = "${var.zone}"

  security_group_ids = ["xxxx", "xxx"]
}

provider "exoscale" {
  key = "${var.key}"
  secret = "${var.secret}"
  compute_endpoint = "https://api.exoscale.com/compute"
}
