provider "template" {
  version = "~> 2.1"
}

provider "exoscale" {
  version = "~> 0.15"
  key = var.key
  secret = var.secret
}

data "exoscale_compute_template" "node" {
  zone = var.zone
  name = var.template
}

resource "exoscale_affinity" "rke" {
  name = "rke-nodes"
  description = "keep nodes of different hosts"
}

resource "exoscale_security_group" "rke" {
  name = "rke"
}

// https://rancher.com/docs/rancher/v2.x/en/installation/requirements/
resource "exoscale_security_group_rules" "rke" {
  security_group_id = exoscale_security_group.rke.id

  ingress {
    protocol = "TCP"
    cidr_list = ["0.0.0.0/0", "::/0"]
    ports = ["22", "80", "443", "2376", "6443", "30000-32767"]
  }

  ingress {
    protocol = "UDP"
    cidr_list = ["0.0.0.0/0", "::/0"]
    ports = ["30000-32767"]
  }

  ingress {
    protocol = "TCP"
    user_security_group_list = [exoscale_security_group.rke.name]
    ports = ["2379-2380", "4789", "10250-10252", "10256"]
  }

  ingress {
    protocol = "UDP"
    user_security_group_list = [exoscale_security_group.rke.name]
    ports = ["8472", "30000-32767"]
  }

}

resource "exoscale_compute" "node" {
  count = length(var.hostnames)
  display_name = element(var.hostnames, count.index)
  template_id = data.exoscale_compute_template.node.id
  zone = var.zone
  size = "Medium"
  disk_size = 50

  key_pair = var.key_pair
  affinity_groups = [exoscale_affinity.rke.name]
  security_groups = ["default", exoscale_security_group.rke.name]

  user_data = element(data.template_cloudinit_config.config.*.rendered, count.index)

  tags = {
    managedby = "terraform"
  }
}

output "master_ips" {
  value = join(",", formatlist("%s@%s", exoscale_compute.node.*.username, exoscale_compute.node.*.ip_address))
}
