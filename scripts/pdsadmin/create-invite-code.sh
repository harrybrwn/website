#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

if [ -z "${PDS_HOSTNAME:-}" ]; then
  echo "Error: PDS_HOSTNAME is required" 1>&2
  exit 1
elif [ -z "${PDS_ADMIN_PASSWORD:-}" ]; then
  echo "Error: PDS_ADMIN_PASSWORD is required" 1>&2
  exit 1
fi

curl \
  --fail \
  --silent \
  --show-error \
  --request POST \
  --user "admin:${PDS_ADMIN_PASSWORD}" \
  --header "Content-Type: application/json" \
  --data '{"useCount": 1}' \
  "https://${PDS_HOSTNAME}/xrpc/com.atproto.server.createInviteCode" | jq --raw-output '.code'
