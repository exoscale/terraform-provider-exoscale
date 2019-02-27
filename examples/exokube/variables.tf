variable "key" {}
variable "secret" {}
variable "key_pair" {}

variable "zone" {
  default = "ch-dk-2"
}

variable "template" {
  default = "Linux Ubuntu 16.04 LTS 64-bit"
}

variable "ubuntu_flavor" {
  default = "xenial"
}

variable "docker_version" {
  default = "18.06.2~ce~3-0~ubuntu"
}

variable "calico_version" {
  default = "3.5"
}
