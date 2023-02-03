terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.31.0"
    }
  }
}

resource "cloudflare_record" "domain_verification_record" {
  for_each = {
    "${var.em_id}"  = "u25288590.wl091.sendgrid.net"
    "s1._domainkey" = "s1.domainkey.u25288590.wl091.sendgrid.net"
    "s2._domainkey" = "s2.domainkey.u25288590.wl091.sendgrid.net"
  }
  name    = each.key
  value   = each.value
  type    = "CNAME"
  ttl     = var.ttl
  proxied = var.proxied
  zone_id = var.zone_id
}