# SOS Bucket Policies

This example demonstrates how to manage Exoscale [SOS Bucket Policies](https://community.exoscale.com/documentation/storage/bucketpolicy/).

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_bucket_uri = <<EOT
https://sos-ch-gva-2.exo.io/my-bucket-6bed6744-c98e-aaba-1710-3ac09522348e

EOT
my_data_policy = "{\"default-service-strategy\":\"allow\",\"services\":{\"sos\":{\"type\":\"allow\"}}}"
```
