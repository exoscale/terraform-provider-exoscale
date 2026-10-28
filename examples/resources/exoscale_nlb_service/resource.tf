resource "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}

resource "exoscale_nlb_service" "my_nlb_service" {
  nlb_id = exoscale_nlb.my_nlb.id
  zone   = exoscale_nlb.my_nlb.zone
  name   = "my-nlb-service"

  instance_pool_id = exoscale_instance_pool.my_instance_pool.id
  protocol         = "tcp"
  port             = 443
  target_port      = 8443
  strategy         = "round-robin"

  healthcheck {
    mode     = "https"
    port     = 8443
    uri      = "/healthz"
    tls_sni  = "example.net"
    interval = 5
    timeout  = 3
    retries  = 1
  }
}

# The backends can also be picked by instance labels, in which case the
# the instance pool can be omitted.
resource "exoscale_nlb_service" "my_nlb_labels_service" {
  nlb_id      = exoscale_nlb.my_nlb.id
  zone        = exoscale_nlb.my_nlb.zone
  name        = "my-nlb-labels-service"
  protocol    = "tcp"
  port        = 8053
  target_port = 8053

  label_selector = {
    role                      = "web"
    "exoscale.com/managed-by" = "terraform"
  }

  healthcheck {
    port = 8053
  }
}
