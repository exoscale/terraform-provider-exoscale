data "exoscale_sks_nodepool_template" "my_template" {
  zone         = "ch-gva-2"
  kube_version = "1.34"
  variant      = "nvidia"
}

output "my_template_id" {
  value = data.exoscale_sks_nodepool_template.my_template.id
}
