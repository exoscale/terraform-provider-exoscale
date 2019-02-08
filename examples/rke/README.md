# HA Kubernetes installation using RKE

Inspired by [High Availability Installation][ha] from the Rancher 2.0 documentation.

## Prerequisites

You need a SSH keypair, [to be installed in your account][ssh] and report its name into `terraform.tfvars`.

```ini
key = "EXO..."
secret = "..."
key_pair = "my-key"
```

## Provision Linux Hosts

The configuration sets up three hosts in Frankfurt.

```
$ terraform apply

...

master_ips = ubuntu@89.145.160.57,ubuntu@89.145.160.47,ubuntu@89.145.160.24
```

## Before jumping to the official tutorial

Aside from the SSL keys (either self-signed or signed by CA), we can infer the
`nodes` section of the [`cluster.yml`](./cluster.yml.example) file from the
Terraform output.

```
# the SSH keypair created in the prerequisites and referenced as `my-key` above.
ssh_key_path: ~/.ssh/id_rsa
nodes:
 - address: 89.145.160.57
   user: ubuntu
   role: [controlplane,etcd,worker]
 - address: 89.145.160.47
   user: ubuntu
   role: [controlplane,etcd,worker]
 - address: 89.145.160.24
   user: ubuntu
   role: [controlplane,etcd,worker]

# ...
```

Continue with the [RKE installation and cluster setup][step4].

[ssh]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[ha]: https://rancher.com/docs/rancher/v2.x/en/installation/ha-server-install/
[step4]: https://rancher.com/docs/rancher/v2.x/en/installation/ha-server-install/#4-download-rke
