#!/bin/sh

set -eu

harrybrwn_cert() {
	echo "\$ sudo certbot certonly ..."
	sudo certbot certonly \
		--dns-cloudflare    \
		--dns-cloudflare-credentials /root/.secrets/harrybrwn-cloudflare.ini \
		--dns-cloudflare-propagation-seconds 15 \
		-d 'harrybrwn.com' \
		-d '*.harrybrwn.com'
}

hryb_cert() {
	echo "\$ sudo certbot certonly ..."
	sudo certbot certonly \
		--dns-cloudflare    \
		--dns-cloudflare-credentials /root/.secrets/harrybrwn-cloudflare.ini \
		--dns-cloudflare-propagation-seconds 15 \
		-d 'hryb.dev'          \
		-d '*.hryb.dev'        \
		-d '*.api.hryb.dev'    \
		-d '*.rpc.hryb.dev'    \
		-d '*.grpc.hryb.dev'   \
		-d '*.s3.hryb.dev'     \
		-d '*.db.hryb.dev'     \
		-d '*.rdb.hryb.dev'    \
		-d '*.ldap.hryb.dev'   \
		-d '*.saml.hryb.dev'   \
		-d '*.radius.hryb.dev' \
		-d '*.pkg.hryb.dev'    \
		-d '*.git.hryb.dev'    \
		-d '*.registry.hryb.dev'
}

echo '$ sudo -v'
sudo -v

if sudo test ! -d /etc/letsencrypt/live/harrybrwn.com; then
	harrybrwn_cert
fi
if sudo test ! -d /etc/letsencrypt/live/hryb.dev; then
	hryb_cert
fi
