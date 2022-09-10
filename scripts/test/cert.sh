#!/bin/bash

set -e

declare -r CADIR="./ca"
declare -r KEY_SIZE=2048
export OPENSSL_CONF=./openssl.cnf
declare override=false

ca() {
  local serial=./ca/serial
  if ! ${override} && [ -f ca/ca.crt ] && [ -f "${serial}" ] && [ -f ca/index.txt ]; then
    return 0
  else
    rm -r ca certs issued
  fi
  mkdir -p ca certs issued
  echo '' > ./ca/index.txt
  if [ ! -f "${serial}" ]; then echo '1000' > ./ca/serial; fi
  openssl req \
    -new -x509 \
    -subj '/CN=test_ca/O=HarryBrown/OU=Personal Site' \
    -extensions v3_ca \
    -out ca/ca.crt \
    -keyout ca/ca.key \
    -nodes
}

server() {
  local name="${1:-server}"
  local key="issued/${name}.key"
  local csr="issued/${name}.csr"
  local crt="issued/${name}.crt"
  if ! ${override}; then echo override; fi
  if [ -f "${key}" ]; then echo key; fi
  if [ -f "${crt}" ]; then echo crt; fi
  if ! ${override} && [ -f "${key}" ] && [ -f "${crt}" ]; then
    return 0
  fi
  openssl req \
    -new      \
    -subj '/CN=server/O=HarryBrown' \
    -keyout "${key}"    \
    -out "${csr}"       \
    -nodes
  openssl ca           \
    -batch             \
    -keyfile ca/ca.key \
    -in "${csr}"       \
    -out "${crt}"
  rm "${csr}"
}

main() {
  while [ $# -gt 0 ]; do
    case $1 in
      --override)
        override=true
        shift 1
        ;;
    esac
  done

  ca
  server 'test'
}

main "$@"