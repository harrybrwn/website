#!/bin/bash

set -e

declare -r CADIR="./ca"

main() {
	if [ ! -d "${CADIR}" ]; then
		mkdir "${CADIR}"
	fi
	openssl genrsa -out "${CADIR}/ca.key" 2048
	openssl req -new -x509   \
		-key "${CADIR}/ca.key" \
		-out "${CADIR}/ca.crt" \
		-extensions v3_ca
}

main