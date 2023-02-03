variable "zone_id" {
  type        = string
  description = "Cloudflare zone ID."
}

variable "em_id" {
	type = string
}

variable "sendgrid_subdomain" {
	type = string
	description = "Subdomain of sendgrid verification values"
}

variable "ttl" {
  type        = number
  description = "DNS record ttl."
  default     = 1
}

variable "proxied" {
  type        = bool
  description = "Turn on the cloudflare proxy for the email domsins."
  default     = false
}
