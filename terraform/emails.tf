# TODO change name "hrry_me" to "account_email_dst"
resource "cloudflare_email_routing_address" "hrry_me" {
  account_id = var.cf_account_id
  email      = var.destination_email
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
      "bsky",
      "h3rie+bsky",
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

resource "cloudflare_email_routing_rule" "hrry_me" {
  for_each = toset([
    "h",
    "harry",
    "admin",
    "bsky",
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
