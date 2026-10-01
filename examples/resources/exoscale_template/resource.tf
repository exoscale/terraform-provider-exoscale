resource "exoscale_template" "my_template" {
  zone             = "ch-gva-2"
  name             = "my-custom-template"
  description      = "My custom template"
  url              = "https://example.com/my-image.qcow2"
  checksum         = "8a4c47..."
  default_user     = "root"
  boot_mode        = "uefi"
  password_enabled = false
  ssh_key_enabled  = true
}
