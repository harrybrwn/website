resource "cloudflare_record" "minecraft_server" {
  name    = "mc"
  value   = var.gateway_ip
  type    = "A"
  proxied = false
  ttl     = 120 # seconds
  comment = "Created by terraform."
  zone_id = local.zones.hrry_lol
}

resource "cloudflare_record" "minecraft_server_local" {
  name    = "local.mc"
  value   = "10.0.0.14"
  type    = "A"
  proxied = false
  ttl     = 120 # seconds
  comment = "Created by terraform."
  zone_id = local.zones.hrry_lol
}
