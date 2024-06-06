#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v

CONF="${1:-}"
NAME=""
if [ -z "${CONF}" ]; then
  CONF="$(terraform output --raw config)"
  NAME=openvpn
fi
NAME="$(basename ${CONF} | sed -Ee 's/\..*$//g;')"

sudo nmcli connection import type openvpn file "${CONF}"
#sudo nmcli connection modify "${NAME}" ipv4.dns '10.0.0.1' # Set correct router IP (this may change)
sudo nmcli connection modify "${NAME}" ipv6.method 'disabled' # disable ipv6
sudo systemctl restart NetworkManager.service

echo "Config imported. You may need to change the DNS settings!"
