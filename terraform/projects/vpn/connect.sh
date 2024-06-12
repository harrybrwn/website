#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v
CONF="${1:-}"
if [ -z "${CONF}" ]; then
  CONF="$(tofu output --raw config)"
fi

echo "Disabling IPv6"
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1
sudo sysctl -w net.ipv6.conf.default.disable_ipv6=1

echo "Connecting with ${CONF}"
sudo openvpn --config "${CONF}"
