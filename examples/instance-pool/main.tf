# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone     = "ch-gva-2"
  my_template = "Linux Ubuntu 22.04 LTS 64-bit"
}

# Existing resources (<-> data sources)
data "exoscale_template" "my_template" {
  zone = local.my_zone
  name = local.my_template
}

data "exoscale_security_group" "default" {
  name = "default"
}

# SSH
# -> ssh.tf

# VPC
resource "exoscale_vpc" "my_vpc" {
  zone = local.my_zone
  name = "my-vpc"
}

resource "exoscale_vpc_subnet" "my_vpc_subnet" {
  zone       = local.my_zone
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "my-vpc-subnet"
  ipv4_block = "10.0.0.0/24"
}

resource "exoscale_vpc_subnet" "my_other_vpc_subnet" {
  zone       = local.my_zone
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "my-other-vpc-subnet"
  ipv4_block = "10.0.1.0/24"
}

resource "exoscale_vpc_subnet" "subnet2" {
  zone       = local.my_zone
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "subnet2"
  ipv4_block = "10.0.2.0/24"
}

# Sample instance pool
resource "exoscale_instance_pool" "my_instance_pool" {
  zone = local.my_zone
  name = "my-instance-pool"

  template_id   = data.exoscale_template.my_template.id
  instance_type = "standard.medium"
  disk_size     = 10
  size          = 3

  key_pair = exoscale_ssh_key.my_ssh_key.name

  security_group_ids = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.my_ssh_security_group.id,
  ]

  vpc {
    id = exoscale_vpc.my_vpc.id

    subnet_ids = [
      exoscale_vpc_subnet.my_vpc_subnet.id,
      exoscale_vpc_subnet.subnet2.id,
      # exoscale_vpc_subnet.my_other_vpc_subnet.id,
    ]
  }
}

# Outputs
output "ssh_connection" {
  value = join("\n", formatlist(
    "ssh -i id_ssh %s@%s  # %s",
    data.exoscale_template.my_template.default_user,
    exoscale_instance_pool.my_instance_pool.instances.*.public_ip_address,
    exoscale_instance_pool.my_instance_pool.instances.*.name,
  ))
}
