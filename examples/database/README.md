# Database

This example demonstrates how to setup and run Grafana [DBaaS service](https://community.exoscale.com/documentation/dbaas/)
using `exoscale_database` resource, as well as read connection URL using `exoscale_database_uri` datasource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

database_uri = <sensitive>
```

Because database connection URI is sensitive, it can be printed with a command `terraform output`:

```console
terraform output database_uri
"https://my-database-exoscale-<uuid>.aivencloud.com:443"
```
