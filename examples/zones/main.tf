data "exoscale_zones" "example_zones" {}

# Outputs
output "zones_output" {
  value = data.exoscale_zones.example_zones.zones
}
