#!/bin/bash

set -e

readonly DIR="$(pwd)/$(dirname ${BASH_SOURCE[0]})"
source "$DIR/shell/common.sh"

if ! has_certutil; then
	echo "Error: 'certutil' command not found"
	exit 1
fi

if ! grep 'harrybrwn.local' /etc/hosts > /dev/null 2>&1; then
	echo "Local DNS configuration needed:"
	echo "Run the following:"
	echo ' $ echo "127.0.0.1 harrybrwn.local\n127.0.0.1 home.harrybrwn.local" sudo tee -a /etc/hosts'
	echo
	exit 1
fi

if ! certutil -L -d "$CERTDB" -n "$LOCAL_CERT_NAME" > /dev/null 2>&1; then
	echo "Error: Local certificate '$LOCAL_CERT_NAME' not installed!"
	exit 1
fi

if [ ! -f config/pki/certs/ca.crt ] || [ ! -f config/pki/certs/harrybrwn.local/server.key ]; then
  echo "Error: local certificate not generated."
  echo "Please run:"
  echo " $ scripts/certs.sh"
  echo
  exit 1
fi

echo "Looks good."
