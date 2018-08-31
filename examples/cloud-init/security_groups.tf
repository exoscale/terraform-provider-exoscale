resource "exoscale_security_group" "swarm" {
  name = "docker-swarm"
}

resource "exoscale_security_group_rule" "docker_client" {
  security_group_id = "${exoscale_security_group.swarm.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"
  start_port = 2376
  end_port = 2376
}

resource "exoscale_security_group_rule" "docker_swarm" {
  security_group_id = "${exoscale_security_group.swarm.id}"
  protocol = "TCP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.swarm.id}"
  start_port = 2377
  end_port = 2377
}

resource "exoscale_security_group_rule" "docker_swarm_nodes_tcp" {
  security_group_id = "${exoscale_security_group.swarm.id}"
  protocol = "TCP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.swarm.id}"
  start_port = 7946
  end_port = 7946
}

resource "exoscale_security_group_rule" "docker_swarm_nodes_udp" {
  security_group_id = "${exoscale_security_group.swarm.id}"
  protocol = "UDP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.swarm.id}"
  start_port = 7946
  end_port = 7946
}

resource "exoscale_security_group_rule" "docker_swarm_overlay_net" {
  security_group_id = "${exoscale_security_group.swarm.id}"
  protocol = "UDP"
  type = "INGRESS"
  user_security_group_id = "${exoscale_security_group.swarm.id}"
  start_port = 4789
  end_port = 4789
}
