data "exoscale_kms_key_store" "my_key_store" {
  zone = "ch-gva-2"
  name = "my-key-store"
}

output "my_key_store_id" {
  value = data.exoscale_kms_key_store.my_key_store.id
}
