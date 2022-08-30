variable "cloudflare_token" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

variable "gateway_ip" {
  description = "IP address of the main gateway."
  type = string
}

variable "cloudflare_api_key" { default = "" }
variable "cloudflare_email" { default = "" }
