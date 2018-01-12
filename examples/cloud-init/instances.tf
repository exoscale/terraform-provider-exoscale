resource "exoscale_affinity" "swarm_manager" {
  name = "docker-swarm-manager"
  description = "keep the swarm managers on different hypervisors"
}

resource "exoscale_compute" "master" {
  count = "${var.master}"
  display_name = "master-${count.index}"
  template = "${var.template}"
  zone = "${var.zone}"
  size = "Medium"
  disk_size = 50

  key_pair = "${var.key_pair}"
  affinity_groups = ["${exoscale_affinity.swarm_manager.name}"]
  security_groups = ["default", "${exoscale_security_group.swarm.name}"]

  user_data = "${data.template_file.cloud_init.rendered}"
}

output "master_ips" {
  value = "${join(",", exoscale_compute.master.*.ip_address)}"
}
