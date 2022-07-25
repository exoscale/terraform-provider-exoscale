# Simple Object Storage (SOS)

This example demonstrates how to manage
[Simple Object Storage (SOS)](https://community.exoscale.com/documentation/storage/)
buckets and objects, using the stock
[S3/AWS](https://registry.terraform.io/providers/hashicorp/aws/) provider.

```console
$ terraform init
$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_object_uri = "https://sos-ch-gva-2.exo.io/my-bucket-2da17217-8ef3-254d-429e-08bced1109a5/my-object.txt"

$ wget -qO- https://sos-ch-gva-2.exo.io/my-bucket-2da17217-8ef3-254d-429e-08bced1109a5/my-object.txt
Hello World!
```
