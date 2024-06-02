#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v

CONF="${1:-}"
if [ -z "${CONF}" ]; then
  CONF="$(terraform output --raw config)"
fi

sudo nmcli connection import type openvpn file "${CONF}"

echo "Config imported. You may need to change the DNS settings!"
