variable "zone" {
  default = "de-fra-1"
}

variable "template" {
  default = "Linux Ubuntu 18.04 LTS 64-bit"
}

data "exoscale_compute_template" "website" {
  zone = var.zone
  name = var.template
}

resource "exoscale_instance_pool" "website" {
  name = "instancepool-website"
  description = "test"
  template_id = data.exoscale_compute_template.website.id
  service_offering = "medium"
  size = 3
  zone = var.zone
}

resource "exoscale_nlb" "website" {
  name = "website"
  description = "This is the Network Load Balancer for my website"
  zone = var.zone
}

resource "exoscale_nlb_service" "website" {
  zone = exoscale_nlb.website.zone
  name = "website"
  description = "This is the Network Load Balancer Service for my website"
  nlb_id = exoscale_nlb.website.id
  instance_pool_id = exoscale_instance_pool.website.id
	protocol = "tcp"
	port = 9595
	target_port = 9595
	strategy = "round-robin"

  healthcheck {
    mode = "tcp"
    port = 9595
    interval = 5
    timeout = 5
    retries = 1
    uri = ""
  }
}

provider "exoscale" {
  key = var.key
  secret = var.secret
}
