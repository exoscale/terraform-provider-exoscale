# Multipart Cloud-Init

> « [Cloud-ini](http://cloudinit.readthedocs.io/) is the _defacto_ multi-distribution package that handles early initialization of a cloud instance.

This example builds a multiple machine with tailored cloud-init config files from [a template file](cloud-init.yml.tpl). This example relies on the [`template_cloudinit_config`](https://www.terraform.io/docs/providers/template/d/cloudinit_config.html) (see [data.tf](data.tf)) to also send a shell script file along.

## Usage

```console
$ terraform apply

Outputs:

hostnames = ubuntu@159.100.251.212, ubuntu@159.100.251.199
```

We may now connect to one of them and check that everything when fine.

```
$ ssh ubuntu@159.100.251.212
$ tail /var/log/cloud-config-output.log
...
Setting up jq (1.5+dfsg-2) ...

Setup via o:Terraform v0.11.5

Cloud-init v. 17.2 running 'modules:final' at Wed, 28 Mar 2018 10:20:51 +0000. Up 29.02 seconds.
Cloud-init v. 17.2 finished at Wed, 28 Mar 2018 10:21:41 +0000. Datasource DataSourceCloudStack.  Up 78.19 seconds

$ jq -V
jq-1.5-1-a5b5cbe
```

`jq` has been installed, and a shell script showing up the Terraform version was used to build it also ran during the cloud-init run.
