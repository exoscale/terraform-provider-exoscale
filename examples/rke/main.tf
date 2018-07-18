provider "template" {
  version = "~> 1.0"
}

provider "exoscale" {
  version = "~> 0.9.22"
  key = "${var.key}"
  secret = "${var.secret}"
}

resource "exoscale_affinity" "rke" {
  name = "rke-nodes"
  description = "keep nodes of different hosts"
}

resource "exoscale_security_group" "rke" {
  name = "rke"
}

resource "exoscale_security_group_rule" "rke_ssh" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 22
  end_port = 22
}

resource "exoscale_security_group_rule" "rke_docker" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.rke.id}"
  start_port = 2378
  end_port = 2380
}

resource "exoscale_security_group_rule" "rke_api_server" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.rke.id}"
  start_port = 6443
  end_port = 6443
}

resource "exoscale_security_group_rule" "rke_external_api_server" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 6443
  end_port = 6443
}

resource "exoscale_security_group_rule" "rke_external_http" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 80
  end_port = 80
}

resource "exoscale_security_group_rule" "rke_external_tls" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 443
  end_port = 443
}

resource "exoscale_security_group_rule" "rke_logs_metrics" {
  security_group_id = "${exoscale_security_group.rke.id}"
  protocol = "TCP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.rke.id}"
  start_port = 10250
  end_port = 10250
}


resource "exoscale_compute" "node" {
  count = "${length(var.hostnames)}"
  display_name = "${element(var.hostnames, count.index)}"
  template = "${var.template}"
  zone = "${var.zone}"
  size = "Medium"
  disk_size = 50

  key_pair = "${var.key_pair}"
  affinity_groups = ["${exoscale_affinity.rke.name}"]
  security_groups = ["default", "${exoscale_security_group.rke.name}"]

  user_data = "${element(data.template_cloudinit_config.config.*.rendered, count.index)}"

  tags {
    managedby = "terraform"
  }
}

output "master_ips" {
  value = "${join(",", formatlist("%s@%s", exoscale_compute.node.*.username, exoscale_compute.node.*.ip_address))}"
}
