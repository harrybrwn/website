#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v

CONF="${1:-}"
if [ -z "${CONF}" ]; then
  CONF="$(tofu output --raw config)"
fi
NAME="$(basename ${CONF} | sed -Ee 's/\..*$//g;')"

PAGER='' nmcli connection show "${NAME}" > /dev/null
if [ "$?" != 10 ]; then
  nmcli connection delete "${NAME}"
fi

sudo nmcli connection import type openvpn file "${CONF}"
#sudo nmcli connection modify "${NAME}" ipv4.dns '10.0.0.1' # Set correct router IP (this may change)
sudo nmcli connection modify "${NAME}" ipv6.method 'disabled' # disable ipv6
sudo systemctl restart NetworkManager.service

echo "Config imported. You may need to change the DNS settings!"
