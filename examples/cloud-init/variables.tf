variable "token" {}
variable "secret" {}
variable "key_pair" {}


variable "master" {
  default = 3
}

// hostnames are used as a source
variable "hostnames" {
  type = "list"
  default = [
    "huey",
    "dewey",
    "louie"
  ]
}

variable "zone" {
  default = "ch-dk-2"
}

variable "template" {
  default = "Linux Ubuntu 17.10 64-bit"
}

