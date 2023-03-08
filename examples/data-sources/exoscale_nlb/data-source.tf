data "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}

output "my_nlb_id" {
  value = data.exoscale_nlb.my_nlb.id
}
