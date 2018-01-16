resource "exoscale_network" "intra" {
  name = "intra"
  display_text = "hello world"
  zone = "ch-dk-2"
  network_offering = "PrivNet"
}

resource "exoscale_compute" "machine" {
  # ...
}

resource "exoscale_nic" "" {
  network_id = "${exoscale_network.intra.id}"
  compute_id = "${exoscale_compute.machine.id}"
}
