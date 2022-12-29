resource "cloudflare_zone" "hrry_io" {
  account_id = var.cf_account_id
  zone       = "hrry.io"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone_settings_override" "hrryio_settings" {
  zone_id = cloudflare_zone.hrry_io.id
  settings {
    always_online            = "off"
    automatic_https_rewrites = "off"
    max_upload = 100 # megabytes (default 100, any bigger is behind a paywall)
    minify {
      css  = "off"
      js   = "off"
      html = "off"
    }
    # Ingress does SNI routing. So make origin requests using ssl
    ssl = "strict"
  }
}

resource "cloudflare_zone_dnssec" "hrry_io_dnssec" {
  zone_id = cloudflare_zone.hrry_io.id
}

resource "cloudflare_record" "hrry_io_dns" {
  for_each = toset([
    "cr",
    "ip",
  ])
  name    = each.key
  value   = "_homelab.${cloudflare_zone.hrry_io.zone}"
  type    = "CNAME"
  proxied = true
  ttl     = 1
  zone_id = cloudflare_zone.hrry_io.id
}

resource "cloudflare_record" "hrry_io_github_pages" {
  name    = "web"
  value   = "harrybrwn.github.io"
  type    = "CNAME"
  proxied = false
  ttl     = 120
  zone_id = cloudflare_zone.hrry_io.id
}
  
