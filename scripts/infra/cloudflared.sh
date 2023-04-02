#!/bin/sh

set -eu

docker container run \
	--name cloudflared \
	--restart always \
	-it cloudflare/cloudflare:latest \
		tunnel --no-autoupdate \
		run --token "$@"
