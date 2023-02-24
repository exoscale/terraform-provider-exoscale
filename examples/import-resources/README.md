# Importing existing resources

Based on the [weblog post: Managing your existing infrastructure with
Terraform](https://www.exoscale.com/syslog/managing-your-existing-infrastructure-with-terraform/)

This example demonstrates how to start using Terraform along an already deployed infrastructure and
import existing resources - e.g. compute instance and its security groups - to match a newly-written
Terraform configuration.

## Preliminary setup

Create a compute instance in the [Portal](https://portal.exoscale.com/):
  - Hostname: `my-instance`
  - Region: `ch-gva-2`
  - OS Template: `Linux Ubuntu 22.04 LTS 64-bit`
  - Type: `Small`
  - Disk: `10 GB`

And initialize your Terraform configuration:

``` console
terraform init
```

## Using Exoscale CLI

Likewise, using the [Exoscale CLI](https://github.com/exoscale/cli/) instead:

```console
$ exo compute instance create \
  --zone 'ch-gva-2' \
  --template 'Linux Ubuntu 22.04 LTS 64-bit' \
  --instance-type 'standard.medium' \
  --disk-size 10 \
  'my-instance'
```

Which also comes handy to identify existing resources **IDs**
(required to perform the imports below):

``` console
$ exo compute instance list | grep -Fw my-instance
┼──────────────────────────────────────┼─────────────┼──────────┼─────────────────┼───────────────┼─────────┼
│                  ID                  │    NAME     │   ZONE   │      TYPE       │  IP ADDRESS   │  STATE  │
┼──────────────────────────────────────┼─────────────┼──────────┼─────────────────┼───────────────┼─────────┼
│ 66afd436-5243-4bdb-8929-963d81bf7325 │ my-instance │ ch-gva-2 │ standard.medium │ 185.19.28.210 │ running │
┼──────────────────────────────────────┼─────────────┼──────────┼─────────────────┼───────────────┼─────────┼

$ exo compute security-group list | grep -Fw default
┼──────────────────────────────────────┼─────────┼
│                  ID                  │  NAME   │
┼──────────────────────────────────────┼─────────┼
│ dd31c3cd-e19d-47d4-b187-914386bc0303 │ default │
┼──────────────────────────────────────┼─────────┼

```

## Import the security group

```console
terraform import \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET \
  exoscale_security_group.default dd31c3cd-e19d-47d4-b187-914386bc0303

...

exoscale_security_group.default: Importing from ID "dd31c3cd-e19d-47d4-b187-914386bc0303"...

...

Import successful!
```

## Import the compute instance

```console
terraform import \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET \
  exoscale_compute_instance.my_instance 66afd436-5243-4bdb-8929-963d81bf7325@ch-gva-2

...

exoscale_compute_instance.my_instance: Importing from ID "66afd436-5243-4bdb-8929-963d81bf7325@ch-gva-2"...

...

Import successful!
```

(mark the `@<zone>` stanza required when specifying the instance ID)
