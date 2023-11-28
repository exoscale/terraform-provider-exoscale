# IAM API Key SOS

This example demonstrates how to provision
[IAM](https://community.exoscale.com/documentation/iam/iam-api-key-roles-policies)
Roles and API Keys using `exoscale_iam_role` and `exoscale_iam_api_key` resources.

In our configuration we will define 2 Roles:
- Read-Write access for 2 different buckets,
- Read-Only access for one specific bucket.

For each role we define an API Key and assigne the role.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

sos_rw_key = "EXO238a7fc2c438815bc71e9bc3"
sos_rw_secret = <sensitive>
sos_ro_key = "EXO2ebe95a7ddee20c608b7faf6"
sos_ro_secret = <sensitive>
```

Because API Secret is sensitive, it can be printed with a command `terraform output`:

```console
terraform output sos_rw_key
"ABC..."
```
