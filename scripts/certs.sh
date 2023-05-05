#!/bin/bash

set -e

# shellcheck disable=SC2155
readonly DIR="$(pwd)/$(dirname "${BASH_SOURCE[0]}")"
# shellcheck source=scripts/shell/common.sh
source "$DIR/shell/common.sh"

declare -r ORG="HarryBrown"

declare -r PKI="./config/pki"
declare -r CA_KEY="${PKI}/certs/ca.key"
declare -r CA_CRT="${PKI}/certs/ca.crt"

export OPENSSL_CONF=./config/openssl.cnf

uninstall_cert() {
	local cert_name="${1}"
	if [ -z "${cert_name}" ]; then
		echo "no cert name to uninstall"
		return 1
	fi
	if ! has_certutil; then
		echo "Error: 'certutil' command not found"
		return 1
	fi
	if certutil -L -d "${CERTDB}" -n "${cert_name}" > /dev/null 2>&1; then
		certutil -D -d "${CERTDB}" -n "${cert_name}"
	else
		echo "could not find certificate installation \"${cert_name}\""
	fi
}

load_cert() {
	local cert_name="${1}"
	if [ -z "${cert_name}" ]; then
		echo "no cert name to load"
		return 1
	fi
	local cert_file="${2}"
	if [ ! -f "${cert_file}" ]; then
		echo "cert file \"${cert_file}\" does not exist"
		return 1
	fi
	if ! has_certutil; then
		echo "Error: 'certutil' command not found"
		return 1
	fi

	uninstall_cert "${cert_name}"
	certutil -A -n "${cert_name}" \
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
		-sha256 -days 720
}

cert_created() {
	local certs="${PKI}/certs"
	local CN=""
	while [ $# -gt 0 ]; do
		case $1 in
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
	crt="${certs}/${CN}.crt"
	key="${certs}/${CN}.key"
	if [ ! -f "${crt}" ] || [ ! -f "${key}" ]; then
		return 1
	fi
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
	csr="${certs}/${CN}.csr"
	crt="${certs}/${CN}.crt"
	key="${certs}/${CN}.key"
	ext="${certs}/${CN}.ext"
	#[ ! -d "${certs}/${CN}" ] && mkdir -p "${certs}/${CN}"

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
	rm -f "${csr}" "${ext}" "${PKI}/certs/ca.srl"
}

usage() {
	echo "Usage"
	echo "  certs.sh [flags...]"
	echo
	echo "Flags"
	echo "  -h, --help          show this help message"
	echo "      --no-install    skip the certificate installation step"
	echo "      --only-install  only install existing certificates"
	echo "      --check         check that certificates have been created"
	echo
}

# Flags
INSTALL=true
ONLY_INSTALL=false
CHECK=false

while [ $# -gt 0 ]; do
	case $1 in
		-h|--help)
			usage
			exit
			;;
		--check)
			CHECK=true
			shift
			;;
		--no-install)
			INSTALL=false
			shift
			;;
		--only-install)
			INSTALL=true
			ONLY_INSTALL=true
			shift
			;;
		*)
			echo "Error: unknown flag: \"$1\""
			exit 1
			;;
	esac
done

if ${CHECK}; then
	cert_created -cn "harrybrwn.com"
	cert_created -cn "hrry.me"
	cert_created -cn "hrry.dev"
	exit 0
fi

if ! ${ONLY_INSTALL}; then
	rm -rf "${PKI}/certs"
	mkdir -p "${PKI}/certs"

	ca_cert "harrybrwn local dev"
	server_cert -cn "harrybrwn.com" \
		-alt "harrybrwn.local"     -alt "*.harrybrwn.local" \
		-alt 'harrybrwn.com-local' -alt '*.harrybrwn.com-local'
	server_cert -cn 'hrry.me' \
		-alt 'hrry.local'    -alt '*.hrry.local' \
		-alt 'hrry.me-local' -alt '*.hrry.me-local'
	server_cert -cn 'hrry.dev' \
		-alt 'hrry.local'     -alt '*.hrry.local' \
		-alt 'hrry.dev-local' -alt '*.hrry.dev-local'
	server_cert -cn 'hydra' -alt 'auth.hrry.local'

	# ln -s "harrybrwn.local" "${PKI}/certs/harrybrwn.com"
fi

uninstall_cert "${LOCAL_CERT_NAME}" || true

if $INSTALL; then
	load_cert "${LOCAL_CERT_NAME}" "${CA_CRT}"
	# sudo cp "${CA_CRT}" /usr/local/share/ca-certificates/harrybrwn.crt
	# sudo update-ca-certificates --fresh
fi

K8S="config/k8s/dev/certs"
CERTS="${PKI}/certs"
rm -rf "${K8S}" && mkdir -p "${K8S}"

cp ${CERTS}/* "${K8S}"
