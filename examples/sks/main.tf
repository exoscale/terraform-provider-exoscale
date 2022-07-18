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

# Sample anti-affinity group
resource "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name = "my-anti-affinity-group"
}

# Sample SKS cluster
resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = local.my_zone
  name = "my-sks-cluster"
}

resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  zone       = local.my_zone
  cluster_id = exoscale_sks_cluster.my_sks_cluster.id
  name       = "my-sks-nodepool"

  instance_type = "standard.small"
  size          = 3

  anti_affinity_group_ids = [
    exoscale_anti_affinity_group.my_anti_affinity_group.id,
  ]
  security_group_ids = [
    data.exoscale_security_group.default.id,
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
