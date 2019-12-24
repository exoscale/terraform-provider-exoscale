output "hostnames" {
  value = join(", ", formatlist("%s@%s", exoscale_compute.test.*.username, exoscale_compute.test.*.ip_address))
}
