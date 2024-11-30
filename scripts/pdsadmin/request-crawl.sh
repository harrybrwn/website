#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

PDS_HOSTNAME=""
# Env files
ENV_FILES=()
ARGS=()

# Parse global flags
while [ $# -gt 0 ]; do
  case $1 in
    -h|-help|--help|help)
      echo "Usage..."
      exit 0
      ;;
    -env|--env)
      ENV_FILES+=("${2}")
      shift 2
      ;;
    -hostname|--hostname)
      PDS_HOSTNAME="$2"
      shift 2
      ;;
    -admin-password|--admin-password)
      PDS_ADMIN_PASSWORD="$2"
      shirt 2
      ;;
    *)
      echo "Error: unkown flag \"$1\"" 1>&2
      exit 1
      ;;
  esac
done

for file in ${ENV_FILES[@]}; do
  source "$file"
done

if [ -z "${PDS_HOSTNAME}" ]; then
  echo "Error: PDS_HOSTNAME is required" 1>&2
  exit 1
fi
if [ -z "${PDS_CRALWERS}" ]; then
  PDS_CRAWLERS=https://bsky.network
fi

set -- "${ARGS[@]}"

RELAY_HOSTS="${1:-}"
if [[ "${RELAY_HOSTS}" == "" ]]; then
  RELAY_HOSTS="${PDS_CRAWLERS}"
fi

if [[ "${RELAY_HOSTS}" == "" ]]; then
  echo "ERROR: missing RELAY HOST parameter." >/dev/stderr
  echo "Usage: $0 <RELAY HOST>[,<RELAY HOST>,...]" >/dev/stderr
  exit 1
fi

for host in ${RELAY_HOSTS//,/ }; do
  echo "Requesting crawl from ${host}"
  if [[ $host != https:* && $host != http:* ]]; then
    host="https://${host}"
  fi
  curl \
    --fail \
    --silent \
    --show-error \
    --request POST \
    --header "Content-Type: application/json" \
    --data "{\"hostname\": \"${PDS_HOSTNAME}\"}" \
    "${host}/xrpc/com.atproto.sync.requestCrawl" >/dev/null
done

echo "done"
