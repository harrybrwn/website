terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.16"
    }
    # https://registry.terraform.io/providers/cloudflare/cloudflare/latest/docs
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.0"
    }
    # https://registry.terraform.io/providers/grafana/grafana/latest/docs
    grafana = {
      source = "grafana/grafana"
      version = "~> 1.27.0"
    }
  }
}

provider "aws" {
  profile = "default"
  region  = "us-west-1"
}

provider "grafana" {
  url = "https://grafana.hrry.dev"
  auth = ""
}
