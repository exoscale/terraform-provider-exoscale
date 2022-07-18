# Providers
# -> providers.tf

# Customizable parameters
locals {
  my_zone   = "ch-gva-2"
  my_bucket = "my-bucket"
  my_object = "my-object.txt"
}

# Sample random UUID
resource "random_uuid" "my_uuid" {
}

# SOS bucket
resource "aws_s3_bucket" "my_bucket" {
  bucket = "${local.my_bucket}-${resource.random_uuid.my_uuid.result}"

  lifecycle {
    ignore_changes = [
      object_lock_configuration,
    ]
  }
}

# (associated ACL)
resource "aws_s3_bucket_acl" "my_bucket_acl" {
  bucket = aws_s3_bucket.my_bucket.bucket

  acl = "public-read"
}

# (associated CORS)
resource "aws_s3_bucket_cors_configuration" "my_bucket_cors" {
  bucket = aws_s3_bucket.my_bucket.bucket

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://s3-website-test.hashicorp.com"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

# SOS object (file)
resource "aws_s3_object" "my_object" {
  bucket = aws_s3_bucket.my_bucket.bucket

  key    = local.my_object
  source = local.my_object
  acl    = "public-read"
  etag   = filemd5(local.my_object)
}

# Outputs
output "my_object_uri" {
  value = format(
    "https://sos-%s.exo.io/%s/%s",
    aws_s3_bucket.my_bucket.region,
    aws_s3_bucket.my_bucket.bucket,
    aws_s3_object.my_object.key,
  )
}
