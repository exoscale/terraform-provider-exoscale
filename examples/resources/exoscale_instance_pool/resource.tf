data "exoscale_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

resource "exoscale_instance_pool" "my_instance_pool" {
  zone = "ch-gva-2"
  name = "my-instance-pool"

  template_id   = data.exoscale_template.my_template.id
  instance_type = "standard.medium"
  disk_size     = 10
  size          = 3
}
