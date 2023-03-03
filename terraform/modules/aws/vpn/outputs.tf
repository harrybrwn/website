output "public_ip" {
	value = aws_eip.vpn_ip.public_ip
}

output "config_file" {
  value = "${local.ovpn_config_path}/openvpn.ovpn"
}