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
#   comment = "Created by terraform."
#   zone_id = cloudflare_zone.h3y_sh.id
# }

resource "cloudflare_email_routing_settings" "h3y" {
  zone_id = cloudflare_zone.h3y_sh.id
  enabled = "true"
}

resource "cloudflare_email_routing_rule" "h3y" {
  for_each = toset(concat(
    [
      "br3ie_+twitch0", # my twitch account lol
      "oof",
      "private",
      "me",
      "x+h6n",
      "patreon",
    ],
  ))
  zone_id = cloudflare_zone.h3y_sh.id
  enabled = true
  name    = "cf email route ${each.key}"
  matcher {
    type  = "literal"
    field = "to"
    value = "${each.key}@${cloudflare_zone.h3y_sh.zone}"
  }
  action {
    type  = "forward"
    value = [var.destination_email]
  }
}
