data "exoscale_kms_key_store" "my_key_store" {
  zone = "ch-gva-2"
  name = "my-key-store"
}

resource "exoscale_kms_key" "my_external_key" {
  zone = "ch-gva-2"
  name = "my-external-key"

  xks = {
    key_store_id    = data.exoscale_kms_key_store.my_key_store.id
    external_key_id = "<external-key-id>"
  }
}
