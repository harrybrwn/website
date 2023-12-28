#!/bin/bash

set -euo pipefail

key() {
  if [ ! -f "$1" ]; then
    openssl genrsa -out "$1" 4096
  fi
}

if [ ! -f ca.srl ]; then
  echo "obase=16; $RANDOM" | bc > ca.srl
fi

key ca.key
openssl req      \
  -new -x509     \
  -nodes -sha256 \
  -key "ca.key"  \
  -out "ca.crt"  \
  -days 720      \
  -subj "/OU=development/O=hrry.dev Security"

key saml.key
openssl req -new \
  -subj "/OU=development/O=hrry.dev Security/CN=saml.hrry.local" \
  -key saml.key \
  -out saml.csr
openssl x509 -req \
  -in saml.csr    \
  -CA ca.crt      \
  -CAkey ca.key   \
  -out saml.crt   \
  -extfile saml.cnf
rm saml.csr
