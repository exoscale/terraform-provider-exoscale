# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone = "ch-gva-2"
}

# Existing resources (<-> data sources)
data "exoscale_security_group" "default" {
  name = "default"
}

# Sample SKS cluster
resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone         = local.my_zone
  name         = "my-sks-cluster"
  auto_upgrade = true
  exoscale_csi = true
  # The default cni is "calico", use the Cilium security group rules provided below if
  # you change the cni to "cilium"
  # cni = "cilium"
}

# (ad-hoc anti-affinity group)
resource "exoscale_anti_affinity_group" "my_sks_anti_affinity_group" {
  name = "my-sks-anti-affinity-group"
}

# (ad-hoc security group)
resource "exoscale_security_group" "my_sks_security_group" {
  name = "my-sks-security-group"
}

resource "exoscale_security_group_rule" "kubelet" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Kubelet"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 10250
  end_port          = 10250
  # (beetwen worker nodes only)
  user_security_group_id = exoscale_security_group.my_sks_security_group.id
}

# mandatory rules for Calico CNI (default)
resource "exoscale_security_group_rule" "calico_vxlan" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "VXLAN (Calico)"
  type              = "INGRESS"
  protocol          = "UDP"
  start_port        = 4789
  end_port          = 4789
  # (beetwen worker nodes only)
  user_security_group_id = exoscale_security_group.my_sks_security_group.id
}

# mandatory rules for Cilium CNI (default)
# resource "exoscale_security_group_rule" "cilium_icmp_health" {
#   security_group_id = exoscale_security_group.my_sks_security_group.id
#   description       = "Cilium ICMP healthcheck"
#   type              = "INGRESS"
#   protocol          = "ICMP"
#   icmp_type         = 8
#   icmp_code         = 0
#   # (beetwen worker nodes only)
#   user_security_group_id = exoscale_security_group.my_sks_security_group.id
# }

# resource "exoscale_security_group_rule" "cilium_vxlan" {
#   security_group_id = exoscale_security_group.my_sks_security_group.id
#   description       = "VXLan (Cilium)"
#   type              = "INGRESS"
#   protocol          = "UDP"
#   start_port        = 8472
#   end_port          = 8472
#   # (beetwen worker nodes only)
#   user_security_group_id = exoscale_security_group.my_sks_security_group.id
# }

# resource "exoscale_security_group_rule" "cilium_udp_health" {
#   security_group_id = exoscale_security_group.my_sks_security_group.id
#   description       = "Cilium UDP healthcheck"
#   type              = "INGRESS"
#   protocol          = "UDP"
#   start_port        = 4240
#   end_port          = 4240
#   # (beetwen worker nodes only)
#   user_security_group_id = exoscale_security_group.my_sks_security_group.id
# }

resource "exoscale_security_group_rule" "nodeport_tcp" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Nodeport TCP services"
  type              = "INGRESS"
  protocol          = "TCP"
  start_port        = 30000
  end_port          = 32767
  # (public)
  cidr = "0.0.0.0/0"
}

resource "exoscale_security_group_rule" "nodeport_udp" {
  security_group_id = exoscale_security_group.my_sks_security_group.id
  description       = "Nodeport UDP services"
  type              = "INGRESS"
  protocol          = "UDP"
  start_port        = 30000
  end_port          = 32767
  # (public)
  cidr = "0.0.0.0/0"
}

# (worker nodes)
resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  zone       = local.my_zone
  cluster_id = exoscale_sks_cluster.my_sks_cluster.id
  name       = "my-sks-nodepool"

  instance_type = "standard.medium"
  size          = 3

  anti_affinity_group_ids = [
    exoscale_anti_affinity_group.my_sks_anti_affinity_group.id,
  ]
  security_group_ids = [
    data.exoscale_security_group.default.id,
    resource.exoscale_security_group.my_sks_security_group.id,
  ]
}

# (administration credentials)
resource "exoscale_sks_kubeconfig" "my_sks_kubeconfig" {
  zone       = local.my_zone
  cluster_id = exoscale_sks_cluster.my_sks_cluster.id

  user   = "kubernetes-admin"
  groups = ["system:masters"]

  ttl_seconds           = 3600
  early_renewal_seconds = 300
}

resource "local_sensitive_file" "my_sks_kubeconfig_file" {
  filename        = "kubeconfig"
  content         = exoscale_sks_kubeconfig.my_sks_kubeconfig.kubeconfig
  file_permission = "0600"
}

# Outputs
output "my_sks_cluster_endpoint" {
  value = exoscale_sks_cluster.my_sks_cluster.endpoint
}

output "my_sks_kubeconfig" {
  value = local_sensitive_file.my_sks_kubeconfig_file.filename
}

output "my_sks_connection" {
  value = format(
    "export KUBECONFIG=%s; kubectl cluster-info; kubectl get pods -A",
    local_sensitive_file.my_sks_kubeconfig_file.filename,
  )
}
