variable "github_username" {
  type        = string
  description = "Github username."
}

variable "zone_id" {
  type        = string
  description = "Cloudflare zone ID."
}

variable "domain_verify_code" {
  type        = string
  description = "DNS verification challenge code for Github Pages."
}

variable "ttl" {
  type        = number
  default     = 1 # 1 is used as "auto"
  description = "DNS record TTL"
}
