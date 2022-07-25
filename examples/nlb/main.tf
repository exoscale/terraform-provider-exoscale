# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone     = "ch-gva-2"
  my_template = "Linux Ubuntu 22.04 LTS 64-bit"
}

# Existing resources (<-> data sources)
data "exoscale_compute_template" "my_template" {
  zone = local.my_zone
  name = local.my_template
}

data "exoscale_security_group" "default" {
  name = "default"
}

# SSH
# -> ssh.tf

# Sample security group
resource "exoscale_security_group" "my_http_security_group" {
  name = "my-http-security-group"
}

resource "exoscale_security_group_rule" "http" {
  security_group_id = exoscale_security_group.my_http_security_group.id
  description       = "HTTP"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 80
  end_port          = 80
  cidr              = "0.0.0.0/0"
}

# Sample instance pool
# (hosting the target service)
resource "exoscale_instance_pool" "my_instance_pool" {
  zone = local.my_zone
  name = "my-instance-pool"

  template_id   = data.exoscale_compute_template.my_template.id
  instance_type = "standard.medium"
  disk_size     = 10
  size          = 3

  key_pair  = exoscale_ssh_key.my_ssh_key.name
  user_data = file("cloud-config.yaml")

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
    exoscale_security_group.my_http_security_group.id,
  ]
}

# Sample network load-balancer (NLB)
resource "exoscale_nlb" "my_nlb" {
  zone = local.my_zone
  name = "my-nlb"
}

resource "exoscale_nlb_service" "my_nlb_service" {
  zone = local.my_zone
  name = "my-nlb-service"

  nlb_id      = exoscale_nlb.my_nlb.id
  protocol    = "tcp"
  port        = 80
  target_port = 80
  strategy    = "round-robin"

  healthcheck {
    mode     = "http"
    port     = 80
    interval = 5
    timeout  = 3
    retries  = 2
    uri      = "/"
  }

  instance_pool_id = exoscale_instance_pool.my_instance_pool.id
}

# Outputs
output "ssh_connection" {
  value = join("\n", formatlist(
    "ssh -i id_ssh %s@%s  # %s",
    data.exoscale_compute_template.my_template.username,
    exoscale_instance_pool.my_instance_pool.instances.*.public_ip_address,
    exoscale_instance_pool.my_instance_pool.instances.*.name,
  ))
}

output "my_nlb_ip_address" {
  value = exoscale_nlb.my_nlb.ip_address
}

output "my_nlb_service" {
  value = format(
    "http://%s:80",
    exoscale_nlb.my_nlb.ip_address,
  )
}
