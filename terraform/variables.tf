variable "cloudflare_token" {
  type = string
}

variable "cloudflare_account_id" {
  type = string
}

variable "gateway_ip" {
  description = "IP address of the main gateway."
  type        = string
}

variable "staging_ip" {
  description = "Local IP address of staging environment's gateway machine."
  type        = string
}

variable private_ip {
  description = "Local IP address of staging environment's gateway machine."
  type        = string
}

variable postgres_password {
  type = string
}