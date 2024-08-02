resource "cloudflare_record" "hrry_dev_dns" {
  for_each = toset([
    "files",
    "gopkg",
    "hooks",
    "ip",
    "grafana",
    "auth",
  ])
  name    = each.key
  value   = "_homelab.hrry.dev"
  type    = "CNAME"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
  comment = "Created by terraform."
  zone_id = local.zones.hrry_dev
}

resource "cloudflare_record" "hrry_dev_private" {
  for_each = toset([
    "s3-console",
    "console.s3",
    "s3",
  ])
  name    = each.key
  value   = var.private_gateway_ip
  type    = "A"
  proxied = false
  ttl     = 60
  comment = "Created by terraform."
  zone_id = local.zones.hrry_dev
}

# Staging DNS records
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
  comment = "Created by terraform."
  zone_id = local.zones.hrry_dev
}

# resource "cloudflare_r2_bucket" "apt" {
#   name       = "apt"
#   location   = "wnam"
#   account_id = var.cf_account_id
# }
