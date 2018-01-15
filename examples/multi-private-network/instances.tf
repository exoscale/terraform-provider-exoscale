resource "exoscale_network" "intra" {
  name = "intra"
  display_text = "hello world"
  zone = "ch-dk-2"
  network_offering = "PrivNet"
}
