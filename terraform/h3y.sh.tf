resource "cloudflare_zone" "h3y_sh" {
  account_id = var.cf_account_id
  zone       = "h3y.sh"
  type       = "full"
  plan       = "free"
}

resource "cloudflare_zone_dnssec" "h3y_sh_dnssec" {
  zone_id = cloudflare_zone.h3y_sh.id
}

module "github_page" {
  source             = "./github-page"
  github_username    = "harrybrwn"
  zone_id            = cloudflare_zone.h3y_sh.id
  ttl                = 60
  domain_verify_code = var.gh_pages_verify_domain_code
}
