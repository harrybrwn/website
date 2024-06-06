#!/usr/bin/env bash

set -euo pipefail
set -x
curl -O https://raw.githubusercontent.com/angristan/openvpn-install/master/openvpn-install.sh
chmod +x openvpn-install.sh

# Set client password preference.
#   1 - no password
#   2 - prompt user for a password when generating a client
export PASS=1

# DNS:
#   1  - system default
#   2  - self hosted resolver
#   3  - cloudflare
#   4  - quad9
#   5  - quad9 uncensored
#   6  - FDN
#   7  - DNS.WATCH
#   8  - OpenDNS
#   9  - Google
#   10 - Yandex basic
#   11 - Adguard
#   12 - NextDNS
#   13 - Custom DNS (input from propmt or $DNS1 and $DNS2)
export DNS=3

# Port Choice
#   1 - use 1194
#   2 - read a port number from prompt or $PORT
#   3 - random port
export PORT_CHOICE=1

# Protocol
#   1 - udp
#   2 - tcp
export PROTOCOL_CHOICE=1

# Approve IP automatically (y/n)
export APPROVE_IP=y

if [[ -z "${public_ip}" ]]; then
  export IP="${public_ip}"
fi

sudo AUTO_INSTALL=y \
    APPROVE_INSTALL=y \
    IPV6_SUPPORT=n \
    CLIENT=${client} \
    ./openvpn-install.sh

# vim: ft=sh
