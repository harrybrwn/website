resource "cloudflare_zone" "willmusic_education" {
  account_id = var.cf_account_id
  zone       = "willmusic.education"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone_settings_override" "willmusic_education_settings" {
  zone_id = cloudflare_zone.willmusic_education.id
  settings {
    automatic_https_rewrites = "off"
    ssl                      = "strict"
  }
}

resource "cloudflare_zone_dnssec" "willmusic_education_dnssec" {
  zone_id = cloudflare_zone.willmusic_education.id
}

resource "cloudflare_email_routing_settings" "willmusic_education" {
  zone_id = cloudflare_zone.willmusic_education.id
  enabled = "true"
}

resource "cloudflare_record" "willmusic_education_dns" {
  name    = "@"
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1
  comment = "Created by terraform."
  zone_id = cloudflare_zone.willmusic_education.id
}
