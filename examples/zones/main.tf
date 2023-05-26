data "exoscale_zones" "example_zones" {
  name = "gva"
}

# Outputs
output "zones_output" {
  value = data.exoscale_zones.example_zones.zones
}
