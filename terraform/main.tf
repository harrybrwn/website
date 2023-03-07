terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
    # https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.31.0"
    }
    github = {
      source  = "integrations/github"
      version = "5.13.0"
    }
    # https://registry.terraform.io/providers/grafana/grafana/latest/docs
    grafana = {
      source  = "grafana/grafana"
      version = "~> 1.27.0"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = var.aws_region
}

provider "github" {}

provider "grafana" {
  url  = "https://grafana.hrry.dev"
  auth = ""
}
