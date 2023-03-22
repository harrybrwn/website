#!/bin/sh

set -eu

get_name() {
	case "${1}" in
		linux/arm64)
			echo "aarch64-linux-musl"
			;;
		linux/arm/v7)
			echo 'arm-linux-musleabi'
			;;
		linux/arm/v6)
			echo 'armv6-linux-musleabi'
			;;
		linux/amd64)
			echo 'x86_64-linux-musl'
			;;
	esac
}

if [ -z "${1:-}" ]; then
	echo "Error: no target given" 1>&2
	exit 1
fi

target="${1}"
shift

while [ $# -gt 0 ]; do
	case $1 in
		-h|--help)
			;;
	esac
	shift
done

name=$(get_name "${target}")
curl -fsL "https://more.musl.cc/11.2.1/x86_64-linux-musl/${name}-cross.tgz" -o "/tmp/${name}-cross.tgz"
tar -xzf "/tmp/${name}-cross.tgz" \
	--exclude '*/usr' \
	-C /usr/ \
	--strip-components 1
