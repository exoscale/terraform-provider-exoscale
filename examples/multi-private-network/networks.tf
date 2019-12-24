resource "exoscale_network" "intra" {
  name = "demo-intra"
  display_text = "demo intra privnet"
  zone = var.zone
}
