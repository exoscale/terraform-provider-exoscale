output "exokube_ssh" {
  value = format("%s@%s", exoscale_compute.exokube.username, exoscale_compute.exokube.ip_address)
}

output "exokube_https" {
  value = "https://${exoscale_compute.exokube.ip_address}.xip.io"
}
