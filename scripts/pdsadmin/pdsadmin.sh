#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

ARGS=()
ENV_FILES=()
COMMAND=""

PDS_ENV_FILE="config/env/pdsadmin.env"
ENV_FILES+=("${PDS_ENV_FILE}")

# Ensure the user is root, since it's required for most commands.
#if [[ "${EUID}" -ne 0 ]]; then
#  echo "ERROR: This script must be run as root"
#  exit 1
#fi

while [ $# -gt 0 ]; do
  case "$1" in
    -h|-help|--help|help)
      COMMAND="help"
      shift
      ;;
    -env|--env)
      ENV_FILES+=("${2}")
      shift 2
      ;;
    -hostname|--hostname)
      _PDS_HOSTNAME="$2"
      shift 2
      ;;
    -admin-password|--admin-password)
      _PDS_ADMIN_PASSWORD="$2"
      shirt 2
      ;;
    -*)
      if [ -n "${COMMAND}" ]; then
        ARGS+=("$1")
        shift
      else
        echo "Error: unknown flag \"$1\""
        exit 1
      fi
      ;;
    *)
      if [ -n "${COMMAND}" ]; then
        ARGS+=("${1}")
      else
        COMMAND="${1}"
      fi
      shift
      ;;
  esac
done

if [ -z "${COMMAND:-}" ]; then
  COMMAND="help"
fi

for file in ${ENV_FILES[@]}; do
  source "$file"
done

if [ -n "${_PDS_HOSTNAME:-}" ]; then
  PDS_HOSTNAME="${_PDS_HOSTNAME:-}"
fi
if [ -n "${_PDS_ADMIN_PASSWORD:-}" ]; then
  PDS_ADMIN_PASSWORD="${_PDS_ADMIN_PASSWORD:-}"
fi

export PDS_HOSTNAME
export PDS_ADMIN_PASSWORD

if [[ "${COMMAND}" == "-h" || "${COMMAND}" == "-help" || "${COMMAND}" == "--help" ]]; then
  COMMAND="help"
fi

case "${COMMAND}" in
  health)
    URL="https://${PDS_HOSTNAME}/xrpc/_health"
    echo "GET ${URL}"
    echo
    curl -i -XGET "${URL}"
    echo
    ;;
  *)
    "scripts/pdsadmin/${COMMAND}.sh" "${ARGS[@]}"
    ;;
esac

