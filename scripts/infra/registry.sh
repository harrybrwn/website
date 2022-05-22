#!/bin/sh

set -eu

has_secret() {
	docker secret inspect "$1" > /dev/null 2>&1
}

echo "Running sudo:"
echo "$ sudo -v"
sudo -v

confdir=/etc/docker/registry
if [ ! -d "$confdir" ]; then
	sudo mkdir -p "$confdir"
fi

if [ ! -f "$confdir/htpasswd" ]; then
	echo "Error: no basic auth password setup for docker registry"
	echo
	echo "Create an htpasswd file named $confdir/htpasswd"
	exit 1
fi

certname=registry-server-cert
keyname=registry-server-key
passwd=registry-passwd-file
if ! has_secret "$certname"; then
	sudo cat /etc/docker/server-cert.pem | docker secret create "$certname" -
fi
if ! has_secret "$keyname"; then
	sudo cat /etc/docker/server-key.pem | docker secret create "$keyname" -
fi
if ! has_secret "$passwd"; then
	sudo cat "$confdir/htpasswd" | docker secret create "$passwd" -
fi

if ! docker volume inspect registry > /dev/null 2>&1; then
	docker volume create -d 'local' registry
fi

docker service create  \
	--name registry      \
	--secret "$certname" \
	--secret "$keyname"  \
	-e "REGISTRY_HTTP_ADDR=0.0.0.0:443"                       \
	-e "REGISTRY_HTTP_TLS_CERTIFICATE=/run/secrets/$certname" \
	-e "REGISTRY_HTTP_TLS_KEY=/run/secrets/$keyname"          \
	-e "REGISTRY_AUTH=htpasswd"                               \
	-e "REGISTRY_AUTH_HTPASSWD_PATH=/run/secrets/$passwd"     \
	-e "REGISTRY_AUTH_HTPASSWD_REALM=registry.harybrwn.com"   \
	--mount 'type=volume,src=registry,dst=/var/lib/registry'  \
	--mount "type=bind,src=/etc/docker/registry/htpasswd,dst=/etc/registry/htpasswd" \
	--publish 'published=5000,target=443' \
	--constraint 'node.role==manager' \
	--replicas 1 \
	registry:2
