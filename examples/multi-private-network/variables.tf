variable "token" {}
variable "secret" {}
variable "key_pair" {}

variable "master" {
  default = 3
}

variable "zone" {
  default = "ch-dk-2"
}

variable "template" {
  default = "Linux Ubuntu 17.10 64-bit"
}

