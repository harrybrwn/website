# Grafana
variable "grafana_api_key" {
  type = string
}

variable "discord_webhook_url" {
  type        = string
  description = "Alerting webhook URL."
}

# Cloudflare
variable "cloudflare_token" {
  type = string
}

variable "cf_account_id" {
  type = string
}
