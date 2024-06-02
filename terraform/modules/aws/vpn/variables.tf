variable "project_name" {
  type = string
}

variable "storage_path" {
  type    = string
  default = "openvpn"
}

variable "users" {
  type        = list(string)
  default     = []
  description = "A list of users that will get their own vpn client configurations"
}

/* Networking Variables */

variable "vpc_id" {
  type = string
}

variable "public_subnet_id" {
  type = string
}

/* Instance Variables */

variable "ami" {
  type = string
}

variable "instance_type" {
  type = string
}

variable "key_name" {
  type = string
}

variable "vpc_security_group_ids" {
  type    = list(string)
  default = []
}

variable "kms_key_id" {
  type    = string
  default = null
}

/* SSH */

variable "ssh_port" {
  type    = number
  default = 22
}

variable "ssh_user" {
  type = string
}

variable "ssh_cidr" {
  type    = string
  default = "0.0.0.0/0"
}

variable "public_key_openssh" {
  type = string
}

variable "private_key_openssh" {
  type = string
}
