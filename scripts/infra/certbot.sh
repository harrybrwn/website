#!/bin/sh

set -eu

echo '$ sudo -v'
sudo -v

if [ -z "$1" ]; then
	echo "Error: give a domain name"
fi

case "$1" in
	harrybrwn.com)
		echo "\$ sudo certbot certonly ..."
		sudo certbot certonly \
			--dns-cloudflare    \
			--dns-cloudflare-credentials /root/.secrets/harrybrwn-cloudflare.ini \
			-d 'harrybrwn.com' \
			-d '*.harrybrwn.com'
	;;
	hryb.dev)
		echo "\$ sudo certbot certonly ..."
		sudo certbot certonly \
			--dns-cloudflare    \
			--dns-cloudflare-credentials /root/.secrets/hryb-cloudflare.ini \
			-d 'hryb.dev' \
			-d '*.hryb.dev' \
			-d '*.s3.hryb.dev'
	;;
esac
