# IAM API Key Compute

This example demonstrates how to provision
[IAM](https://community.exoscale.com/documentation/iam/iam-api-key-roles-policies)
Roles and API Keys using `exoscale_iam_role` and `exoscale_iam_api_key` resources.

In our configuration we will restrict access to Compute service to only allow deploying private instances.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_api_key = "EXO238a7fc2c438815bc71e9bc3"
my_api_secret = <sensitive>
```

Because API Secret is sensitive, it can be printed with a command `terraform output`:

```console
terraform output my_api_key
"ABC..."c
```

Now if we use our new API key and  try to create compute instance with a public IP address assignment we will get error:

```console
$ exo c i create my-instance 
 ✔ Creating instance "my-instance"... 0s
error: Post "https://api-ch-dk-2.exoscale.com/v2/instance": invalid request: Forbidden by role policy for compute - A deny rule matched. Rule index: 0
```

But we can create a private instance:

```console
$ exo c i create my-instance --private-instance
✔ Creating instance "my-instance"... 12s
┼──────────────────────┼──────────────────────────────────────┼
│   COMPUTE INSTANCE   │                                      │
┼──────────────────────┼──────────────────────────────────────┼
│ ID                   │ f5bd99f6-fdc3-42e9-aec2-cc95d7e22ac7 │
│ Name                 │ my-instance                          │
│ Creation Date        │ 2023-11-13 12:27:20 +0000 UTC        │
│ Instance Type        │ standard.medium                      │
│ Template             │ Linux Ubuntu 22.04 LTS 64-bit        │
│ Zone                 │ ch-dk-2                              │
│ Anti-Affinity Groups │ n/a                                  │
│ Deploy Target        │ -                                    │
│ Security Groups      │ n/a                                  │
│ Private Instance     │ Yes                                  │
│ Private Networks     │ n/a                                  │
│ Elastic IPs          │ n/a                                  │
│ IP Address           │ -                                    │
│ IPv6 Address         │ -                                    │
│ SSH Key              │ -                                    │
│ Disk Size            │ 50 GiB                               │
│ State                │ running                              │
│ Labels               │ n/a                                  │
│ Reverse DNS          │                                      │
┼──────────────────────┼──────────────────────────────────────┼
```
