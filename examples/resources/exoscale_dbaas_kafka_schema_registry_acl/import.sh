# The ACL entry ID is the `id` listed by `exo x get-dbaas-kafka-acl-config <service> -z <zone>`.
terraform import \
  exoscale_dbaas_kafka_schema_registry_acl.my_app_orders \
  my-kafka/acl5ec257a26ce@ch-gva-2
