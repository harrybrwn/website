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

# resource "cloudflare_email_routing_settings" "hrry_lol" {
#   zone_id = local.zones.hrry_lol
#   enabled = true
# }

# resource "cloudflare_email_routing_rule" "hrry_lol" {
#   for_each = toset([
#     "me",
#   ])
#   zone_id = local.zones.hrry_lol
#   enabled = true
#   name    = "cf email route '${each.key}'"
#   matcher {
#     type  = "literal"
#     field = "to"
#     value = "${each.key}@hrry.lol"
#   }
#   action {
#     type  = "forward"
#     value = [var.destination_email]
#   }
# }
