provider "cloudflare" {
  api_token = var.cloudflare_token
}

data "cloudflare_zones" "hrry_me" {
  filter { name = "hrry.me" }
}

data "cloudflare_zones" "hrry_dev" {
  filter { name = "hrry.dev" }
}

data "cloudflare_zone" "harrybrwn_com" {
  name = "harrybrwn.com"
}

resource "cloudflare_zone" "hryb_dev" {
  account_id = data.cloudflare_zone.harrybrwn_com.account_id
  zone       = "hryb.dev"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone" "hrry_io" {
  account_id = data.cloudflare_zone.harrybrwn_com.account_id
  zone       = "hrry.io"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone_dnssec" "hrry_io_dnssec" {
  zone_id = cloudflare_zone.hrry_io.id
}

resource "cloudflare_zone_settings_override" "hrry_me_settings" {
  zone_id = data.cloudflare_zones.hrry_me.zones[0].id
  settings {
    always_online            = "on"
    automatic_https_rewrites = "on"
    browser_cache_ttl = 24 * 60 * 60 # browser cache in seconds
    # Ingress does SNI routing. So make origin requests using ssl
    ssl = "strict"
  }
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

# homelab's gateway DNS records
resource "cloudflare_record" "homelab_gateway" {
  for_each = toset([
    data.cloudflare_zone.harrybrwn_com.id,
    data.cloudflare_zones.hrry_dev.zones[0].id,
    cloudflare_zone.hrry_io.id,
  ])
  zone_id = each.key
  name    = "_homelab"
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
}

# Root DNS record for each main zones
resource "cloudflare_record" "root_dns" {
  for_each = toset([
    data.cloudflare_zone.harrybrwn_com.id,
    data.cloudflare_zones.hrry_me.zones[0].id,
    cloudflare_zone.hrry_io.id,
  ])
  zone_id = each.key
  name    = "@" # root domain only
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
}

resource "cloudflare_record" "hrry_me_dns" {
  for_each = toset([
    "wiki"
  ])
  name    = each.key
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1
  zone_id = data.cloudflare_zones.hrry_me.zones[0].id
}

resource "cloudflare_record" "hrry_dev_dns" {
  for_each = toset([
    "files",
    "gopkg",
    "hooks",
    "ip",
    "registry",
    "grafana",
    "s3-console",
    "s3",
    "auth",
  ])
  name    = each.key
  value   = "_homelab.hrry.dev"
  type    = "CNAME"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
  zone_id = data.cloudflare_zones.hrry_dev.zones[0].id
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

resource "cloudflare_record" "hrry_dev_dns_staging" {
  for_each = toset([
    "stg",
    "*.stg",
  ])
  name    = each.key
  value   = var.staging_ip
  type    = "A"
  proxied = false
  ttl     = 3600
  zone_id = data.cloudflare_zones.hrry_dev.zones[0].id
}

resource "cloudflare_record" "hrry_me_dns_staging" {
  for_each = toset([
    "stg",
    "*.stg",
  ])
  name    = each.key
  value   = var.staging_ip
  type    = "A"
  proxied = false
  ttl     = 3600
  zone_id = data.cloudflare_zones.hrry_me.zones[0].id
}

resource "cloudflare_email_routing_rule" "harry" {
  for_each = toset([
    "cloudflare-notifications",
    # "harry",
    # "admin",
    "ynvybmvyigvtywlscg",
    "trash",
  ])
  zone_id = data.cloudflare_zone.harrybrwn_com.id
  enabled = true
  name    = "cf email route ${each.key}"

  matcher {
    type  = "literal"
    field = "to"
    value = "${each.key}@${data.cloudflare_zone.harrybrwn_com.name}"
  }

  action {
    type  = "forward"
    value = [var.destination_email]
  }
}
