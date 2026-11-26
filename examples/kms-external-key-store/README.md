# KMS External Key Store (XKS)

This example demonstrates how to keep the key material of a KMS key in your own key manager, using the
`exoscale_kms_key_store` and `exoscale_kms_key` resources. See
[Set Up an External Key Store](https://community.exoscale.com/product/security/kms/how-to/set-up-external-key-store/)
for the XKS proxy requirements.

Terraform 1.11 or later is required: the proxy secret key is a write-only attribute.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET \
  -var xks_proxy_endpoint=https://xks.example.com/kms/xks/v1 \
  -var xks_proxy_access_key=$XKS_PROXY_ACCESS_KEY \
  -var xks_proxy_secret_key=$XKS_PROXY_SECRET_KEY \
  -var xks_external_key_id=<external-key-id>

...

Outputs:

my_external_key_id = "019ef461-c0ae-7c9c-94bd-081e70fe2b08"
```

The KMS key is protected with `prevent_destroy`, so `terraform destroy` refuses to run: remove the `lifecycle`
block first if you really want to destroy it. Destroying the KMS key only schedules its deletion (7 days): until
then, the key store cannot be destroyed.
