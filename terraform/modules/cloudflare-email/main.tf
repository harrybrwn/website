terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.31.0"
    }
  }
}

resource "cloudflare_record" "route1" {
  value = "route1.mx.cloudflare.net"
  type  = "MX"
  data {
    priority = 51
  }
  zone_id = var.zone_id
}

resource "cloudflare_record" "route2" {
  value = "route1.mx.cloudflare.net"
  type  = "MX"
  data {
    priority = 50
  }
  zone_id = var.zone_id
}

resource "cloudflare_record" "route3" {
  value = "route1.mx.cloudflare.net"
  type  = "MX"
  data {
    priority = 49
  }
  zone_id = var.zone_id
}

resource "cloudflare_record" "spf" {
  value   = "v=spf1 include:_spf.mx.cloudflare.net ~all"
  type    = "TXT"
  zone_id = var.zone_id
}
