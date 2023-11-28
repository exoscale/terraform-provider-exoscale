# IAM API Key Compute

This example demonstrates how to provision
[IAM](https://community.exoscale.com/documentation/iam/iam-api-key-roles-policies)
Roles and API Keys using `exoscale_iam_role` and `exoscale_iam_api_key` resources.

In our configuration we define Role that only allows access to a single DBaaS service instance.

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
"ABC..."
```

We can now use our API Key to access dbaas service `my-dbaas-service`:

```console
exo dbaas show my-dbaas-instance
┼───────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼
│   DATABASE SERVICE    │                                                                                                                                            │
┼───────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼
│ Zone                  │ ch-dk-2                                                                                                                                    │
│ Name                  │ my-dbaas-instance                                                                                                                          │
│ Type                  │ pg                                                                                                                                         │
│ Plan                  │ hobbyist-2                                                                                                                                 │
│ Disk Size             │ 8.0 GiB                                                                                                                                    │
│ State                 │ running                                                                                                                                    │
│ Creation Date         │ 2023-11-13 14:58:31 +0000 UTC                                                                                                              │
│ Update Date           │ 2023-11-13 15:59:33 +0000 UTC                                                                                                              │
│ Nodes                 │ 1                                                                                                                                          │
│ Node CPUs             │ 2                                                                                                                                          │
│ Node Memory           │ 2.0 GiB                                                                                                                                    │
│ Termination Protected │ false                                                                                                                                      │
│ Maintenance           │ sunday (00:16:44)                                                                                                                          │
│ Version               │ 15.4                                                                                                                                       │
│ Backup Schedule       │ 19:29                                                                                                                                      │
│ URI                   │ postgres://avnadmin:xxxxx@my-dbaas-instance-exoscale-9f21ec06-ab34-44e1-a8de-1714e4120c91.a.aivencloud.com:21699/defaultdb?sslmode=require │
│ IP Filter             │ 1.2.3.4/32                                                                                                                                 │
│ Components            │                                                                                                                                            │
│                       │   pg          my-dbaas-instance-exoscale-9f21ec06-ab34-44e1-a8de-1714e4120c91.a.aivencloud.com:21699   route:dynamic   usage:primary       │
│                       │   pgbouncer   my-dbaas-instance-exoscale-9f21ec06-ab34-44e1-a8de-1714e4120c91.a.aivencloud.com:21700   route:dynamic   usage:primary       │
│                       │                                                                                                                                            │
│ Users                 │ avnadmin (primary)                                                                                                                         │
┼───────────────────────┼────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼
```

If we try the DBaaS instance that is not `my-dbaas-service` we get error:

```console
$ exo dbaas show my-other-dbaas-instance
error: Get "https://api-ch-dk-2.exoscale.com/v2/dbaas-postgres/my-other-dbaas-instance": invalid request: Forbidden by role policy for dbaas - A deny rule matched. Rule index: 0
```
