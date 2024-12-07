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

# ahasend mail setup
resource "cloudflare_record" "ahasend_mail_hrry_me_cname" {
  for_each = {
    "t.mail"                   = "2de58.setup.ahasend.com"
    "psrp.mail"                = "2d7d9.setup.ahasend.com"
    "ahasend._domainkey.mail"  = "2da0f.setup.ahasend.com"
    "ahasend2._domainkey.mail" = "2de02.setup.ahasend.com"
    mail                       = "2d612.setup.ahasend.com"
  }
  type    = "CNAME"
  name    = each.key
  value   = each.value
  proxied = false
  ttl     = 60
  comment = "Created by terraform."
  zone_id = local.zones.hrry_me
}

resource "cloudflare_record" "ahasend_mail_hrry_me_txt" {
  type    = "TXT"
  name    = "_dmarc.mail"
  value   = "v=DMARC1; p=reject; sp=none; adkim=r; aspf=r;"
  proxied = false
  ttl     = 60
  comment = "Created by terraform."
  zone_id = local.zones.hrry_me
}

