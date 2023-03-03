/* Variables */

variable "name" {
	type = string
}

variable "algorithm" {
  type    = string
  default = "ED25519"
}

variable "keys_dir" {
  type    = string
  default = "./keys"
}

locals {
	prv_path = "${var.keys_dir}/${var.name}"
	pub_path = "${var.keys_dir}/${var.name}.pub"
}

/* Resources */

resource "tls_private_key" "key" {
  algorithm = upper(var.algorithm)
}

resource "local_file" "private_key_file" {
  filename             = local.prv_path
  content              = tls_private_key.key.private_key_openssh
  file_permission      = "0600"
  directory_permission = "0777"
}

resource "local_file" "public_key_file" {
  filename             = local.pub_path
  content              = tls_private_key.key.public_key_openssh
  file_permission      = "0600"
  directory_permission = "0777"
}

/* Outputs */

output "private_key_file" {
	value = local.prv_path
}

output "public_key_file" {
	value = local.pub_path
}

output "private_key" {
	value = tls_private_key.key.private_key_openssh
}

output "public_key" {
	value = tls_private_key.key.public_key_openssh
}