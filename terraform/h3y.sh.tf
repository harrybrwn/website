resource "cloudflare_zone" "h3y_sh" {
  account_id = var.cf_account_id
  zone       = "h3y.sh"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone_dnssec" "h3y_sh_dnssec" {
  zone_id = cloudflare_zone.h3y_sh.id
}

# resource "cloudflare_record" "mastodon_local" {
#   name    = "toots"
#   value   = var.private_ip
#   type    = "A"
#   proxied = false
#   ttl     = 60
#   zone_id = cloudflare_zone.h3y_sh.id
# }
