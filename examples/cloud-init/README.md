# Docker Swarm via Cloud-Config

Create the `terraform.tfvars` file (based on `terraform.tfvars.example`.

The setup uses a [cloud-init][cloud-init].
configuration in `init.tpl` to bootstrap the machines.

```
$ terraform apply

Outputs:

master_ips = 10.0.0.1,10.0.0.2,10.0.0.3
```

The IP address of the machines will be part of the `output` variable
`master_ips`. Log into each one of them and setup the
[Docker Swarm][create-swarm].


```
$ terraform output master_ips

$ ssh ubuntu@...
ubuntu@huey: $ sudo docker swarm init --advertise-addr <ip>
ubuntu@huey: $ sudo docker swarm join-token manager

* copy *

$ ssh ubuntu@...
ubuntu@dewey: $ sudo *paste*

$ ssh ubuntu@...
ubuntu@louie: $ sudo *paste*
ubuntu@louie: $ sudo docker node ls

* lists the three machines *
```

To connect remotely, you'll have to configure the HTTPS endpoint to docker.

[cloud-init]: https://cloudinit.readthedocs.io/en/latest/
[create-swarm]: https://docs.docker.com/engine/swarm/swarm-tutorial/create-swarm/
[https]: https://docs.docker.com/engine/security/https/
