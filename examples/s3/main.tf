provider "aws" {
  version = "~> 2.7"
  access_key = var.key
  secret_key = var.secret

  region = var.zone
  endpoints {
    s3 = "https://sos-${var.zone}.exo.io"
  }

  # Deactivate the AWS specific behaviours
  #
  # https://www.terraform.io/docs/backends/types/s3.html#skip_credentials_validation
  skip_credentials_validation = true
  skip_get_ec2_platforms = true
  skip_requesting_account_id = true
  skip_metadata_api_check = true
  skip_region_validation = true
}

resource "aws_s3_bucket" "testbucket" {
  bucket = var.bucket
  acl = "public-read"

  lifecycle {
    ignore_changes = [object_lock_configuration]
  }

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["PUT", "POST"]
    allowed_origins = ["https://s3-website-test.hashicorp.com"]
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
 }

resource "aws_s3_bucket_object" "testobj" {
  bucket = aws_s3_bucket.testbucket.bucket
  acl = "public-read"
  key = "some-text.txt"
  source = "some-text.txt"
  etag   = md5(file("some-text.txt"))
}
