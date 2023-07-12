#!/bin/sh

set -eu

do_certbot() {
	echo "\$ sudo certbot certonly ..."
	sudo certbot certonly \
		--dns-cloudflare \
		--dns-cloudflare-credentials /root/.secrets/harrybrwn-cloudflare.ini \
		--dns-cloudflare-propagation-seconds 15 \
		"$@"
}

harrybrwn_cert() {
	do_certbot \
		-d 'harrybrwn.com' \
		-d '*.harrybrwn.com'
}

hrybdev_cert() {
	do_certbot \
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

hrryme_cert() {
	do_certbot -d 'hrry.me' -d '*.hrry.me'
}

hrrydev_cert() {
	do_certbot \
		-d 'hrry.dev'          \
		-d '*.hrry.dev'        \
		-d '*.api.hrry.dev'    \
		-d '*.rpc.hrry.dev'    \
		-d '*.grpc.hrry.dev'   \
		-d '*.s3.hrry.dev'     \
		-d '*.db.hrry.dev'     \
		-d '*.rdb.hrry.dev'    \
		-d '*.ldap.hrry.dev'   \
		-d '*.saml.hrry.dev'   \
		-d '*.radius.hrry.dev' \
		-d '*.pkg.hrry.dev'    \
		-d '*.git.hrry.dev'    \
		-d '*.registry.hrry.dev'
}

hrrylol_cert() {
	do_certbot \
		-d 'hrry.lol' \
		-d '*.hrry.lol'
}

certs() {
	echo "Running cerbot for \"$1\""
	case $1 in
		hrry.me)
		 	hrryme_cert
			;;
		hrry.dev)
			hrrydev_cert
			;;
		harrybrwn.com)
		 	harrybrwn_cert
			;;
		hryb.dev)
			hrybdev_cert
			;;
		hrry.lol)
			hrrylol_cert
			;;
		*)
			echo "Error: can't handle domain \"$1\"" 1>&2
			exit 1
			;;
	esac
}

echo '$ sudo -v'
sudo -v

#names="harrybrwn.com hryb.dev hrry.me hrry.dev hrry.lol"
names="$@"
if [ -z "${names}" ]; then
	echo "Error: no names given"
fi

for name in $names; do
	if sudo test ! -d "/etc/letsencrypt/live/${name}"; then
		certs "${name}"
	fi
done
