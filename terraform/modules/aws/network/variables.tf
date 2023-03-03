variable "name" {
  type        = string
  description = "Name of the network."
}

variable "env" {
  type    = string
  default = "prd"
}

variable "vpc_cidr" {
  type = string
}

variable "public_cidr" {
  type = string
}

variable "private_cidr" {
  type = string
}
