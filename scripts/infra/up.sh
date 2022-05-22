#!/bin/bash

set -eu

CONTEXT="harrybrwn"

usage() {
	echo "Usage"
	echo
}

while [ $# -gt 0 ]; do
	case $1 in
	  --help)
			usage
			exit 0
			;;
		--context|-c)
			CONTEXT="$2"
			shift 2
			;;
	esac
done

if [ -z "${CONTEXT}" ]; then
	usage
	echo "Error: no docker context given"
	exit 1
fi

docker --context "${CONTEXT}" stack deploy \
	--compose-file config/docker-compose.infra.yml \
	--with-registry-auth \
	infra
