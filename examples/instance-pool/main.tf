variable "zone" {
  default = "ch-gva-2"
}

variable "template" {
  default = "Linux Ubuntu 18.04 LTS 64-bit"
}

data "exoscale_compute_template" "instancepool" {
  zone = "${var.zone}"
  name = "${var.template}"
}

resource "exoscale_instance_pool" "instancepool-test" {
  name = "terraforminstancepool"
  description = "test"
  template_id = "${data.exoscale_compute_template.instancepool.id}"
  serviceoffering = "Medium"
  size = 5
  user_data = "#cloud-config\npackage_upgrade: true\n"
  key_pair = "test"
  zone = "${var.zone}"

  //security_group_ids = ["xxxx", "xxx"]
}

provider "exoscale" {
  key = "${var.key}"
  secret = "${var.secret}"
  compute_endpoint = "https://api.exoscale.com/compute"
}
