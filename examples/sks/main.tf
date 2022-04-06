locals {
  zone = "de-fra-1"
}

resource "exoscale_security_group" "demo" {
  name = "demo"
}

resource "exoscale_security_group_rule" "demo" {
  for_each = {
    kubelet_logs   = { protocol = "TCP", port = 10250, sg = exoscale_security_group.demo.id },
    calico_vxlan   = { protocol = "UDP", port = 4789, sg = exoscale_security_group.demo.id }
    nodeports_ipv4 = { protocol = "TCP", port = "30000-32767", cidr = "0.0.0.0/0" }
    nodeports_ipv6 = { protocol = "TCP", port = "30000-32767", cidr = "::/0" }
  }

  security_group_id      = exoscale_security_group.demo.id
  protocol               = each.value["protocol"]
  type                   = "INGRESS"
  icmp_type              = try(each.value.icmp_type, null)
  icmp_code              = try(each.value.icmp_code, null)
  start_port             = try(split("-", each.value.port)[0], each.value.port, null)
  end_port               = try(split("-", each.value.port)[1], each.value.port, null)
  user_security_group_id = try(each.value.sg, null)
  cidr                   = try(each.value.cidr, null)
}

resource "exoscale_anti_affinity_group" "demo" {
  name = "demo"
}

resource "exoscale_sks_cluster" "demo" {
  zone    = local.zone
  name    = "demo"
}

resource "exoscale_sks_nodepool" "demo" {
  zone          = local.zone
  cluster_id    = exoscale_sks_cluster.demo.id
  name          = "pool"
  instance_type = "standard.medium"
  size          = 3

  anti_affinity_group_ids = [exoscale_anti_affinity_group.demo.id]
  security_group_ids      = [exoscale_security_group.demo.id]
}

resource "exoscale_sks_kubeconfig" "demo_admin" {
  zone = local.zone

  ttl_seconds = 3600
  early_renewal_seconds = 300
  cluster_id = exoscale_sks_cluster.demo.id
  user = "kubernetes-admin"
  groups = ["system:masters"]
}

resource "local_sensitive_file" "kubeconfig" {
  content = exoscale_sks_kubeconfig.demo_admin.kubeconfig
  filename = "kubeconfig"
  file_permission = "0600"

}
output "kubeconfig" {
  value = exoscale_sks_kubeconfig.demo_admin.kubeconfig
  sensitive = true
}

