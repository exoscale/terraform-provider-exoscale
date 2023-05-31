# Zones

This example demonstrates how to get a list of all zones with the `exoscale_zones` data-source.


```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

zones_output = tolist([
  "ch-gva-2",
  "ch-dk-2",
  "at-vie-1",
  "de-fra-1",
  "bg-sof-1",
  "at-vie-2",
])
```
