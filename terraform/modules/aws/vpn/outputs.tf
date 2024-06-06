output "public_ip" {
  value = aws_eip.vpn_ip.public_ip
}

output "ipv4" {
  value = aws_instance.vpn.public_ip
}

output "ipv6" {
  value = var.ipv6 ? element(aws_instance.vpn.ipv6_addresses, 0) : ""
}

output "config_file" {
  value = "${local.ovpn_config_path}/openvpn.ovpn"
}
