locals {
  r2_domain = "${var.cf_account_id}.r2.cloudflarestorage.com"
}

provider "aws" {
  alias      = "cloudflare"
  access_key = var.r2_access_key_id
  secret_key = var.r2_secret_access_key
  region     = "auto"
  endpoints {
    s3 = "https://${local.r2_domain}"
  }
  skip_credentials_validation = true
  skip_region_validation      = true
  skip_requesting_account_id  = true
  skip_get_ec2_platforms      = true
  s3_use_path_style           = true
}

resource "aws_s3_bucket" "registry-bucket" {
  provider           = aws.cloudflare
  bucket             = "container-registry"
}
