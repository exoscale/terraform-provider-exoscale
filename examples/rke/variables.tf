variable "key" {}
variable "secret" {}
variable "key_pair" {}


// hostnames are used as a source
variable "hostnames" {
  type = list(string)
  default = [
    "huey",
    "dewey",
    "louie"
  ]
}

variable "zone" {
  default = "de-fra-1"
}

variable "template" {
  default = "Linux Ubuntu 16.04 LTS 64-bit"
}

variable "ubuntu-flavor" {
  default = "xenial"
}

variable "docker-version" {
  default = "17.03.3"
}
