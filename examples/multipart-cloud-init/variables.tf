variable "key" {}
variable "secret" {}
variable "key_pair" {}

variable "hostnames" {
  type = list(string)
  default = ["alpha", "beta"]
}

variable "zone" {
  default = "ch-dk-2"
}

variable "template" {
  default = "Linux Ubuntu 18.04 LTS 64-bit"
}

