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
    "api",
  ])
  name    = each.key
  value   = "_homelab.hrry.dev"
  type    = "CNAME"
  proxied = true
  ttl     = 1 # proxied records require ttl of 1
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
  zone_id = local.zones.hrry_dev
}
