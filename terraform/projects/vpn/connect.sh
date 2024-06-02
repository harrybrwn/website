#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v
CONF="${1:-}"
if [ -z "${CONF}" ]; then
  CONF="$(terraform output --raw config)"
fi
echo "Connecting with ${CONF}"
sudo openvpn --config "${CONF}"
