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

echo "Ok."
