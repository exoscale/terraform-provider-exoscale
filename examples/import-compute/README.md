# Importing existing infrastructure

Based on the [weblog post: Managing your existing infrastructure with
Terraform](https://www.exoscale.com/syslog/managing-your-existing-infrastructure-with-terraform/)

## Setup

1. Adapt the `terraform.tfvars.example` to `terraform.tfvars`
2. Create a Compute instance in the console
    - Hostname: ada-lovelace
    - Region: CH-DK-2
    - OS Template: Linux Debian 9 64-bit
    - Type: Tiny
    - Disk: 10 GB

## Import the compute instance

```
$ terraform import exoscale_compute.ada <ID>
```

## Import the security group

```
$ terraform import exoscale_security_group.default <ID>
```
