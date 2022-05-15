#!/bin/sh

set -eu

ips="$(curl -sS https://www.cloudflare.com/ips-v4)"
for ip in $ips; do
	echo "allow $ip;"
done

ips="$(curl -sS https://www.cloudflare.com/ips-v6)"
for ip in $ips; do
	echo "allow $ip;"
done
