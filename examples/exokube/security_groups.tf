resource "exoscale_security_group" "exokube" {
  name = "exokube"
  description = "Exokube Security Group"
}

resource "exoscale_security_group_rules" "exokube" {
  security_group_id = exoscale_security_group.exokube.id

  ingress {
    description = "Ping"
    protocol = "ICMP"
    icmp_type = 8
    icmp_code = 0
    cidr_list = ["0.0.0.0/0"]
  }

  ingress {
    description = "Ping6"
    protocol = "ICMPv6"
    icmp_type = 128
    icmp_code = 0
    cidr_list = ["::/0"]
  }

  ingress {
    description = "SSH"
    protocol = "TCP"
    ports = ["22"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }

  ingress {
    description = "Kubernetes API Server"
    protocol = "TCP"
    ports = ["6443"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }

  ingress {
    description = "NodePort TCP"
    protocol = "TCP"
    ports = ["30000-32767"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }

  ingress {
    description = "NodePort UDP"
    protocol = "UDP"
    ports = ["30000-32767"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }
}
