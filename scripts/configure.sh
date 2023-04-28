#!/bin/bash

set -euo pipefail

readonly DIR="$(pwd)/$(dirname ${BASH_SOURCE[0]})"
source "$DIR/shell/common.sh"

RED="\e[31m"
GREEN="\e[32m"
CYAN="\e[36m"
NOCOL="\e[0m"

info() {
	_log="$1"
	shift
	echo -e "${GREEN}[$_log]${NOCOL} $@"
}

error() {
	_log="$1"
	shift
	echo -e "${RED}[$_log]${NOCOL} $@" 1>&2
}

block() {
	echo
	while read line; do
	 	if [ ${#line} -eq 0 ]; then
		 	echo
		else
			echo -e "    $line"
		fi
	done
	echo
}

TOOLS=(
	# Languages
	go
	rustc
	node
	# Build systems and package management
	yarn
	cargo
	make
	# Infrastructure
	terraform
	kubectl
	k3d
	# Utilities
	jq
	mc
	mockgen
	git
	certutil
	curl
	openssl
)
for tool in "${TOOLS[@]}"; do
	path="$(type -P "${tool}")"
	if [ $? -ne 0 ]; then
		error configure "Cannot find tool \"${tool}\""
		exit 1
	fi
	info configure "Found tool \"${tool}\" -> ${path}"
done

docker_plugins="$(docker system info -f json | jq -r '.ClientInfo.Plugins[].Name')"
for cmd in buildx compose; do
	if ! grep "${cmd}" <<< $docker_plugins > /dev/null; then
		error configure "Docker plugin not found: 'docker ${cmd}' is a required plugin"
		exit 1
	fi
	info configure "Found docker plugin \"${cmd}\""
done

if ! grep 'harrybrwn.local' /etc/hosts > /dev/null 2>&1; then
	error configure "Local DNS configuration needed:"
	echo
	echo "Run the following:"
	echo ' $ echo "127.0.0.1 harrybrwn.local\n127.0.0.1 home.harrybrwn.local" sudo tee -a /etc/hosts'
	echo
	exit 1
fi

if ! certutil -L -d "$CERTDB" -n "$LOCAL_CERT_NAME" > /dev/null 2>&1; then
	error configure "Error: Local certificate '$LOCAL_CERT_NAME' not installed!"
	exit 1
fi

PKI_DIR=config/pki/certs
CERT_NAMES=(
	harrybrwn.com
	hrry.me
	hrry.dev
)
for name in "${CERT_NAMES[@]}"; do
	if [ ! -f "${PKI_DIR}/${name}.crt" ] || [ ! -f "${PKI_DIR}/${name}.key" ]; then
		error configure "Error: local certificate for ${name} not generated."
		block <<-EOF
		Please run:
		$ scripts/certs.sh
		EOF
		exit 1
	else
		info configure "Certificate ${name}.crt OK"
	fi
done

echo -e "${CYAN}Configuration looks good.${NOCOL}"
