#!/bin/bash

set -eu

# Docs: https://docs.docker.com/registry/spec/api/#detail
# Notes:
# - /v2/_catalog
# - /v2/<name>/manifests/<refrence>
# - /v2/<name>/tags/list

CONF_DIR="${DOCKER_CONFIG:-$HOME/.docker}"
CONF_FILE="${CONF_DIR}/config.json"
AUTH="$(cat ${CONF_FILE} | jq -r '.auths | .["10.0.0.11:5000"].auth' | base64 -d)"

curl \
	--user "${AUTH}" \
	--basic          \
	--cacert "${CONF_DIR}/ca.pem" \
	"$@"
