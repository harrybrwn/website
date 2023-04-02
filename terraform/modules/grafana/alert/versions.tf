terraform {
  required_providers {
    # https://registry.terraform.io/providers/grafana/grafana/latest/docs
    grafana = {
      source  = "grafana/grafana"
      version = "~> 1.36.1"
    }
  }
}

