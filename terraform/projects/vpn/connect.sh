#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v
echo "Connecting with $(terraform output --raw config)"
sudo openvpn --config "$(terraform output --raw config)"
