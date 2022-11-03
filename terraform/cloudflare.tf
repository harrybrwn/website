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
  zone = "hryb.dev"
}

resource "cloudflare_record" "homelab_gateway" {
  for_each = toset([
    data.cloudflare_zone.harrybrwn_com.id,
    data.cloudflare_zones.hrry_dev.zones[0].id,
  ])
  zone_id = each.key
  name    = "_homelab"
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
}

resource "cloudflare_record" "root_dns" {
  for_each = toset([
    data.cloudflare_zone.harrybrwn_com.id,
    data.cloudflare_zones.hrry_me.zones[0].id,
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
    "harry",
    "admin",
    "ynvybmvyigvtywlscg",
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
