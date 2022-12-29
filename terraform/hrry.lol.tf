resource "cloudflare_record" "minecraft_server" {
  name    = "mc"
  value   = "10.0.0.68"
  type    = "A"
  proxied = false
  ttl     = 120 # seconds
  zone_id = local.zones.hrry_lol
}