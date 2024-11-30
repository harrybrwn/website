resource "cloudflare_record" "hrrylol_bsky0" {
  name    = "_atproto"
  value   = format("%q", "did=did:plc:kk75hbbi3njzjlkh6vgoe24x")
  type    = "TXT"
  proxied = false
  ttl     = 120
  comment = "Created by terraform"
  zone_id = local.zones.hrry_lol
}

resource "cloudflare_record" "hrryme_bluesky_handle" {
  type    = "TXT"
  name    = "_atproto"
  value   = format("%q", "did=did:plc:kzvsijt4365vidgqv7o6wksi")
  proxied = false
  ttl     = 120
  comment = "Created by terraform"
  zone_id = local.zones.hrry_me
}
