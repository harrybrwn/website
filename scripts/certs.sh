#!/bin/bash

set -e

readonly DIR="$(pwd)/$(dirname ${BASH_SOURCE[0]})"
source "$DIR/shell/common.sh"

declare -r ORG="HarryBrown"

declare -r PKI="./config/pki"
declare -r CA_KEY="${PKI}/certs/ca.key"
declare -r CA_CRT="${PKI}/certs/ca.crt"

export OPENSSL_CONF=./config/openssl.cnf

load_cert() {
	local cert_file="${1}"
	if ! has_certutil; then
		echo "Error: 'certutil' command not found"
		return 1
	fi

	if certutil -L -d "$CERTDB" -n "$LOCAL_CERT_NAME" > /dev/null 2>&1; then
		certutil -D -d "$CERTDB" -n "$LOCAL_CERT_NAME"
	fi
	certutil -A -n "$LOCAL_CERT_NAME" \
		-t "CT,c,c" \
		-d "$CERTDB" \
		-i "${cert_file}"
}

ca_cert() {
	local CN="${1}"
	openssl genrsa -out "${CA_KEY}" 4096
	openssl req \
		-new -x509 -nodes \
		-subj "/CN=${CN}/OU=development/O=${ORG}" \
		-key "${CA_KEY}" \
		-out "${CA_CRT}" \
		-sha256 -days 365
}

server_cert() {
	local certs="${PKI}/certs"
	local CN=""
	local csr=""
	local crt=""
	local key=""
	local ext=""
	local alt_names=()
	while [ $# -gt 0 ]; do
		case $1 in
			-alt)
				alt_names+=("$2")
				shift 2
				;;
			-key)
				key="$2"
				shift 2
				;;
			-cert)
				crt="$2"
				shift 2
				;;
			-cn)
				CN="$2"
				shift 2
				;;
			*)
				echo "Unknown flag \"$1\""
				break
				;;
		esac
	done
	if [ -z "${CN}" ]; then
		echo "Error: no common name given, pass '-cn'"
		return 1
	fi
	csr="${certs}/${CN}/server.csr"
	crt="${certs}/${CN}/server.crt"
	key="${certs}/${CN}/server.key"
	ext="${certs}/${CN}/server.ext"
	[ ! -d "${certs}/${CN}" ] && mkdir -p "${certs}/${CN}"

	cat > "${ext}" <<-EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, nonRepudiation, keyEncipherment, dataEncipherment
subjectAltName = @alt_names

[alt_names]
DNS.1 = $CN
EOF

	local i=2
	for alt_name in "${alt_names[@]}"; do
		echo "DNS.${i} = ${alt_name}" >> "${ext}"
		((i+=1))
	done

	openssl genrsa -out "${key}" 4096
	openssl req -new \
		-subj "/CN=${CN}/OU=development/O=${ORG}" \
		-key "${key}" -out "${csr}"
	openssl x509         \
		-req               \
		-in "${csr}"       \
		-CA "${CA_CRT}"    \
		-CAkey "${CA_KEY}" \
		-CAcreateserial    \
		-out "${crt}"      \
		-extfile "${ext}"  \
		-days 825 -sha256
}

rm -rf "${PKI}"
mkdir -p "${PKI}/certs"

ca_cert "harrybrwn local dev"
server_cert -cn "harrybrwn.local" \
	-alt "www.harrybrwn.local" \
	-alt "home.harrybrwn.local"
load_cert "${CA_CRT}"

ln -s "harrybrwn.local" "${PKI}/certs/harrybrwn.com"