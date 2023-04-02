provider "cloudflare" {
  api_token = var.cloudflare_token
}

data "cloudflare_zones" "all" {
  filter {
    account_id = var.cf_account_id
    status     = "active"
  }
}

locals {
  zones = { for z in data.cloudflare_zones.all.zones : replace(z.name, ".", "_") => z.id }
  local_ips = [
    "10.0.0.11",
    "10.0.0.13",
    "10.0.0.21",
    "10.0.0.22",
    "10.0.0.23",
  ]
}

resource "cloudflare_record" "hrry_dev_local_dns" {
  for_each = merge(
    [for d in ["grafana.lab", "s3.lab"] :
      { for i, ip in local.local_ips :
        "${d}-${i}" => {
          domain  = d
          address = ip
        }
    }]...
  )
  name    = each.value.domain
  value   = each.value.address
  type    = "A"
  proxied = false
  ttl     = 120
  zone_id = local.zones.hrry_dev
}
