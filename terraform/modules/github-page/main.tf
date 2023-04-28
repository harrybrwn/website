terraform {
  required_providers {
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 3.31.0"
    }
  }
}

resource "cloudflare_record" "www" {
  type    = "CNAME"
  name    = "www"
  value   = "${var.github_username}.github.io"
  proxied = false
  ttl     = var.ttl
  zone_id = var.zone_id
}

resource "cloudflare_record" "gh_pages_root_v4" {
  for_each = toset([
    "185.199.108.153",
    "185.199.109.153",
    "185.199.110.153",
    "185.199.111.153",
  ])
  type    = "A"
  name    = var.name
  value   = each.key
  proxied = false
  ttl     = var.ttl
  zone_id = var.zone_id
}

resource "cloudflare_record" "gh_pages_root_v6" {
  for_each = toset([
    "2606:50c0:8000::153",
    "2606:50c0:8001::153",
    "2606:50c0:8002::153",
    "2606:50c0:8003::153",
  ])
  type    = "AAAA"
  name    = "@"
  value   = each.key
  proxied = false
  ttl     = var.ttl
  zone_id = var.zone_id
}

resource "cloudflare_record" "gh_pages_domain_verify" {
  type    = "TXT"
  name    = "_github-pages-challenge-${var.github_username}"
  value   = var.domain_verify_code
  proxied = false
  ttl     = var.ttl
  zone_id = var.zone_id
}
