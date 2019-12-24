resource "exoscale_affinity" "swarm_manager" {
  name = "docker-swarm-manager"
  description = "keep the swarm managers on different hypervisors"
}

data "exoscale_compute_template" "master" {
  zone = var.zone
  name = var.template
}

resource "exoscale_compute" "master" {
  count = length(var.hostnames)
  display_name = element(var.hostnames, count.index)
  template_id = data.exoscale_compute_template.master.id
  zone = var.zone
  size = "Medium"
  disk_size = 50

  key_pair = var.key_pair
  affinity_groups = [exoscale_affinity.swarm_manager.name]
  security_groups = ["default", exoscale_security_group.swarm.name]

  user_data = element(data.template_cloudinit_config.config.*.rendered, count.index)

  tags = {
    managedby = "terraform"
    swarm = "master"
  }
}

output "master_ips" {
  value = join(",", exoscale_compute.master.*.ip_address)
}
