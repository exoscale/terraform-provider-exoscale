# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone   = "ch-gva-2"
  my_bucket = "my-bucket"
}

# Sample random UUID
resource "random_uuid" "my_uuid" {
}

# SOS bucket
resource "aws_s3_bucket" "my_bucket" {
  bucket = "${local.my_bucket}-${resource.random_uuid.my_uuid.result}"
}

resource "exoscale_sos_bucket_policy" "my_policy" {
  bucket = "${local.my_bucket}-${resource.random_uuid.my_uuid.result}"
  policy = templatefile("${path.module}/bucket_policy.json.tpl", {})
  zone   = local.my_zone
}

# data "exoscale_sos_bucket_policy" "my_policy_ds" {
#   bucket = "${local.my_bucket}-${resource.random_uuid.my_uuid.result}"
#   zone   = local.my_zone
# }

# Outputs
output "my_bucket_uri" {
  value = format(
    "https://sos-%s.exo.io/%s\n",
    aws_s3_bucket.my_bucket.region,
    aws_s3_bucket.my_bucket.bucket,
  )
}

# output "my_object_uri" {
# data.exoscale_sos_bucket_policy.my_policy_ds.policy,
