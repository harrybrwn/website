resource "cloudflare_zone_settings_override" "hrry_me_settings" {
  zone_id = local.zones.hrry_me
  settings {
    always_online            = "on"
    automatic_https_rewrites = "on"
    browser_cache_ttl        = 24 * 60 * 60 # browser cache in seconds
    # Ingress does SNI routing. So make origin requests using ssl
    ssl = "strict"
  }
}

resource "cloudflare_zone_dnssec" "hrry_me_dnssec" {
  zone_id = local.zones.hrry_me
}

resource "cloudflare_record" "netlify" {
  type    = "CNAME"
  name    = "@"
  value   = "apex-loadbalancer.netlify.com"
  proxied = false
  ttl     = 60
  zone_id = local.zones.hrry_me
}

resource "cloudflare_record" "netlify_www" {
  type    = "CNAME"
  name    = "www"
  value   = "apex-loadbalancer.netlify.com"
  proxied = false
  ttl     = 60
  zone_id = local.zones.hrry_me
}

resource "cloudflare_record" "hrry_me_dns" {
  for_each = toset([
    "wiki",
    "1k",
    "api",
    "l",
  ])
  name    = each.key
  value   = var.gateway_ip
  type    = "A"
  proxied = true
  ttl     = 1
  zone_id = local.zones.hrry_me
}

# Staging DNS records
resource "cloudflare_record" "hrry_me_dns_staging" {
  for_each = toset([
    "stg",
    "*.stg",
  ])
  name    = each.key
  value   = var.staging_ip
  type    = "A"
  proxied = false
  ttl     = 60
  zone_id = local.zones.hrry_me
}

resource "cloudflare_email_routing_settings" "hrry_me" {
  zone_id = local.zones.hrry_me
  enabled = true
}

resource "cloudflare_email_routing_address" "hrry_me" {
  account_id = var.cf_account_id
  email      = var.destination_email
}

resource "cloudflare_email_routing_rule" "hrry_me" {
  for_each = toset([
    "h",
    "harry",
    "admin",
    "trash",
    "trash0",
    "trash1",
    "trash2",
    "trash3",
    "trash4",
    "trash5",
  ])
  zone_id = local.zones.hrry_me
  enabled = true
  name    = "cf email route '${each.key}'"
  matcher {
    type  = "literal"
    field = "to"
    value = "${each.key}@hrry.me"
  }
  action {
    type  = "forward"
    value = [var.destination_email]
  }
}

#module "github_pages" {
#  source = "./modules/github-page"
#  github_username = "harrybrwn"
#  zone_id  = local.zones.hrry_me
#  name = "@"
#  ttl = 1 # auth ttl
#  domain_verify_code=var.gh_pages_domain_verify_codes.hrry_me
#}

# Github pages domain verification. Remove if you add the github-pages module
# call back in.
resource "cloudflare_record" "gh_pages_domain_verify" {
  type    = "TXT"
  name    = "_github-pages-challenge-harrybrwn"
  value   = var.gh_pages_domain_verify_codes.hrry_me
  proxied = false
  ttl     = 1 # auth ttl
  zone_id = local.zones.hrry_me
}
