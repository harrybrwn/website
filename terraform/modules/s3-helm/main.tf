terraform {
  required_providers {
    aws = {
      source                = "hashicorp/aws"
      version               = "~> 4.16"
      configuration_aliases = [aws.s3]
    }
  }
}

resource "aws_s3_bucket" "helm-repo" {
  provider = aws.s3
  bucket   = var.bucket
}

resource "aws_s3_object" "helm-index" {
  bucket   = aws_s3_bucket.helm-repo.bucket
  key      = "index.yaml"
  acl      = "public-read"
  source   = abspath("${var.base_dir}/index.yaml")
  provider = aws.s3
}

resource "null_resource" "helm-packages" {
  provisioner "local-exec" {
    command = join(" ", [
      "helm", "package",
      join(" ", [
        for f in fileset(var.base_dir, "**/Chart.yaml") :
        dirname(f)
      ]),
      "--destination", var.build_dir,
      "--dependency-update",
    ])
    working_dir = var.base_dir
  }
}

resource "aws_s3_object" "helm-charts" {
  depends_on = [
    null_resource.helm-packages,
  ]
  for_each = fileset(abspath("${var.base_dir}/${var.build_dir}/"), "*.tgz")
  bucket   = aws_s3_bucket.helm-repo.bucket
  key      = each.value
  source   = abspath("${var.base_dir}/${var.build_dir}/${each.value}")
  etag     = filemd5(abspath("${var.base_dir}/${var.build_dir}/${each.value}"))
  acl      = "public-read"
  provider = aws.s3
}
