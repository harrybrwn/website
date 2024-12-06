#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

usage() {
  cat <<HELP
Usage:
  pdsadmin [option...] <command>

Options:
  -h, --help              Print this help message
  -v, --verbose           Print verbose output
      --env <file>        Get variables from an .env file (default is "config/env/pdsadmin.env")
      --hostname <host>   Set the PDS_HOSTNAME
      --admin-password    Set the PDS_ADMIN_PASSWORD

Commands:
  update
    Update to the latest PDS version.
      e.g. pdsadmin update

  account
    list
      List accounts
      e.g. pdsadmin account list
    create <EMAIL> <HANDLE>
      Create a new account
      e.g. pdsadmin account create alice@example.com alice.example.com
    delete <DID>
      Delete an account specified by DID.
      e.g. pdsadmin account delete did:plc:xyz123abc456
    takedown <DID>
      Takedown an account specified by DID.
      e.g. pdsadmin account takedown did:plc:xyz123abc456
    untakedown <DID>
      Remove a takedown from an account specified by DID.
      e.g. pdsadmin account untakedown did:plc:xyz123abc456
    reset-password <DID>
      Reset a password for an account specified by DID.
      e.g. pdsadmin account reset-password did:plc:xyz123abc456

  request-crawl [<RELAY HOST>]
      Request a crawl from a relay host.
      e.g. pdsadmin request-crawl bsky.network

  create-invite-code
    Create a new invite code.
      e.g. pdsadmin create-invite-code

  help
      Display this help information.

HELP
}

VERBOSE=false
SCHEME="${PDSADMIN_SCHEME:-https}"

guard_hostname() {
  if [ -z "${PDS_HOSTNAME:-}" ]; then
    echo "Error: PDS_HOSTNAME is required" 1>&2
    exit 1
  fi
}

guard_pw() {
  if [ -z "${PDS_ADMIN_PASSWORD:-}" ]; then
    echo "Error: PDS_PASSWORD is required" 1>&2
    exit 1
  fi
}

# curl a URL and fail if the request fails.
function curl_cmd_get {
  curl --fail --silent --show-error "$@"
}

# curl a URL and fail if the request fails.
function curl_cmd_post {
  curl --fail --silent --show-error --request POST --header "Content-Type: application/json" "$@"
}

# curl a URL but do not fail if the request fails.
function curl_cmd_post_nofail {
  curl --silent --show-error --request POST --header "Content-Type: application/json" "$@"
}

cmd_health() {
  guard_hostname
  guard_pw
  local URL="${SCHEME}://${PDS_HOSTNAME}/xrpc/_health"
  echo "GET ${URL}"
  echo
  curl -i -XGET "${URL}"
  echo
}

cmd_create_invite_code() {
  guard_hostname
  guard_pw
  curl \
    --fail \
    --silent \
    --show-error \
    --request POST \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --header "Content-Type: application/json" \
    --data '{"useCount": 1}' \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.server.createInviteCode" | jq --raw-output '.code'
}

cmd_request_crawl() {
  local RELAY_HOSTS="${1:-}"
  if [[ "${RELAY_HOSTS}" == "" ]]; then
    RELAY_HOSTS="${PDS_CRAWLERS}"
  fi
  if [ -z "${RELAY_HOSTS:-}" ]; then
    echo "Error: no relay hosts. Pass hostnames arguments or set PDS_CRAWLERS" 1>&2
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
}

cmd_account_list() {
  guard_hostname
  guard_pw
  local DIDS="$(curl_cmd_get \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.sync.listRepos?limit=100" | jq --raw-output '.repos[].did'
  )"
  local OUTPUT='[{"handle":"Handle","email":"Email","did":"DID"}'
  for did in ${DIDS}; do
    local ITEM="$(curl_cmd_get \
      --user "admin:${PDS_ADMIN_PASSWORD}" \
      "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.admin.getAccountInfo?did=${did}"
    )"
    OUTPUT="${OUTPUT},${ITEM}"
  done
  OUTPUT="${OUTPUT}]"
  if $VERBOSE; then
    echo "${OUTPUT}" | jq --raw-output '.'
  else
    echo "${OUTPUT}" | jq --raw-output '.[] | [.handle, .email, .did] | @tsv' | column -t
  fi
}

cmd_account_list_dids() {
  guard_hostname
  local DIDS="$(curl_cmd_get \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.sync.listRepos?limit=100" | jq --raw-output '.repos[].did'
  )"
  for did in ${DIDS}; do
    echo "$did"
  done
}

cmd_account_create() {
  guard_hostname
  guard_pw
  local EMAIL="${2:-}"
  local HANDLE="${3:-}"
  if [[ "${EMAIL}" == "" ]]; then
    read -p "Enter an email address (e.g. alice@${PDS_HOSTNAME}): " EMAIL
  fi
  if [[ "${HANDLE}" == "" ]]; then
    read -p "Enter a handle (e.g. alice.${PDS_HOSTNAME}): " HANDLE
  fi
  if [[ "${EMAIL}" == "" || "${HANDLE}" == "" ]]; then
    echo "ERROR: missing EMAIL and/or HANDLE parameters." >/dev/stderr
    exit 1
  fi
  local PASSWORD="$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-24)"
  local INVITE_CODE="$(curl_cmd_post \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --data '{"useCount": 1}' \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.server.createInviteCode" | jq --raw-output '.code'
  )"
  local RESULT="$(curl_cmd_post_nofail \
    --data "{\"email\":\"${EMAIL}\", \"handle\":\"${HANDLE}\", \"password\":\"${PASSWORD}\", \"inviteCode\":\"${INVITE_CODE}\"}" \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.server.createAccount"
  )"
  local DID="$(echo $RESULT | jq --raw-output '.did')"
  if [[ "${DID}" != did:* ]]; then
    local ERR="$(echo ${RESULT} | jq --raw-output '.message')"
    echo "Error: ${ERR}" 1>&2
    exit 1
  fi
  echo
  echo "Account created successfully!"
  echo "-----------------------------"
  echo "Handle   : ${HANDLE}"
  echo "DID      : ${DID}"
  echo "Password : ${PASSWORD}"
  echo "-----------------------------"
  echo "Save this password, it will not be displayed again."
  echo
}

cmd_account_delete() {
  guard_hostname
  guard_pw
  DID="${2:-}"
  if [[ "${DID}" == "" ]]; then
    echo "ERROR: missing DID parameter." >/dev/stderr
    exit 1
  fi
  if [[ "${DID}" != did:* ]]; then
    echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
    exit 1
  fi
  echo "This action is permanent."
  read -r -p "Are you sure you'd like to delete ${DID}? [y/N] " response
  if [[ ! "${response}" =~ ^([yY][eE][sS]|[yY])$ ]]; then
    exit 0
  fi
  curl_cmd_post \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --data "{\"did\": \"${DID}\"}" \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.admin.deleteAccount" >/dev/null
  echo "${DID} deleted"
}

cmd_account_takedown() {
  guard_hostname
  guard_pw
  DID="${2:-}"
  TAKEDOWN_REF="$(date +%s)"
  if [[ "${DID}" == "" ]]; then
    echo "ERROR: missing DID parameter." >/dev/stderr
    exit 1
  fi
  if [[ "${DID}" != did:* ]]; then
    echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
    exit 1
  fi
  PAYLOAD="$(cat <<EOF
  {
    "subject": {
      "\$type": "com.atproto.admin.defs#repoRef",
      "did": "${DID}"
    },
    "takedown": {
      "applied": true,
      "ref": "${TAKEDOWN_REF}"
    }
  }
EOF
)"
  curl_cmd_post \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --data "${PAYLOAD}" \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateSubjectStatus" >/dev/null
  echo "${DID} taken down"
}

cmd_account_untakedown() {
  guard_hostname
  guard_pw
  DID="${2:-}"
  if [[ "${DID}" == "" ]]; then
    echo "ERROR: missing DID parameter." >/dev/stderr
    exit 1
  fi
  if [[ "${DID}" != did:* ]]; then
    echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
    exit 1
  fi
  PAYLOAD=$(cat <<EOF
{
  "subject": {
    "\$type": "com.atproto.admin.defs#repoRef",
    "did": "${DID}"
  },
  "takedown": {
    "applied": false
  }
}
EOF
)
  curl_cmd_post \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --data "${PAYLOAD}" \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateSubjectStatus" >/dev/null
  echo "${DID} untaken down"
}

cmd_account_reset_password() {
  guard_hostname
  guard_pw
  DID="${2:-}"
  PASSWORD="$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-24)"
  if [[ "${DID}" == "" ]]; then
    echo "ERROR: missing DID parameter." >/dev/stderr
    exit 1
  fi
  if [[ "${DID}" != did:* ]]; then
    echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
    exit 1
  fi
  curl_cmd_post \
    --user "admin:${PDS_ADMIN_PASSWORD}" \
    --data "{ \"did\": \"${DID}\", \"password\": \"${PASSWORD}\" }" \
    "${SCHEME}://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateAccountPassword" >/dev/null
  echo
  echo "Password reset for ${DID}"
  echo "New password: ${PASSWORD}"
  echo
}

ARGS=()
ENV_FILES=()
COMMAND=""
HELP=false

PDS_ENV_FILE="config/env/pdsadmin.env"
if [ -f "${PDS_ENV_FILE}" ]; then
  ENV_FILES+=("${PDS_ENV_FILE}")
fi

# Ensure the user is root, since it's required for most commands.
#if [[ "${EUID}" -ne 0 ]]; then
#  echo "ERROR: This script must be run as root"
#  exit 1
#fi

while [ $# -gt 0 ]; do
  case "$1" in
    -h|-help|--help|help)
      # COMMAND="help"
      HELP=true
      shift
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    --scheme)
      SCHEME="$2"
      shift 2
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
if $HELP; then
  COMMAND="help"
fi

set -- "${ARGS[@]}"

case "${COMMAND}" in
  health)
    cmd_health
    ;;
  create-invite-code)
    cmd_create_invite_code "$@"
    ;;
  request-crawl)
    cmd_request_crawl "$@"
    ;;
  account)
    guard_hostname
    guard_pw
    ARGS=()
    while [ $# -gt 0 ]; do
      case "$1" in
        list)
          cmd_account_list "${ARGS[@]}" "$@"
          break
          ;;
        list-dids)
          cmd_account_list_dids "${ARGS[@]}" "$@"
          break
          ;;
        create)
          cmd_account_create "${ARGS[@]}" "$@"
          break
          ;;
        delete)
          cmd_account_delete "${ARGS[@]}" "$@"
          break
          ;;
        takedown)
          cmd_account_takedown "${ARGS[@]}" "$@"
          break
          ;;
        untakedown)
          cmd_account_untakedown "${ARGS[@]}" "$@"
          break
          ;;
        reset-password)
          cmd_account_reset_password "${ARGS[@]}" "$@"
          break
          ;;
        help|-h|-help|--help)
          cat <<-EOF
Usage
  pdsadmin account <subcommand> [args...]

SubCommands
  list
  create
  delete
  takedown
  untakedown
  reset-password
  list-dids
EOF
          break
          ;;
        *)
          ARGS+=("$1")
          shift
          ;;
      esac
    done
    ;;
  config)
    if [[ "${PDS_ADMIN_PASSWORD:-}" = testlab* ]]; then
      echo "admin password: \"${PDS_ADMIN_PASSWORD:-}\""
    else
      echo "admin password: \"$(echo "${PDS_ADMIN_PASSWORD:-}" | sed 's/./*/g')\""
    fi
    echo "hostname:       \"${PDS_HOSTNAME}\""
    exit 0
    ;;
  help|-h|-help|--help)
    usage
    exit 0
    ;;
  *)
    echo "Error: unknown command \"${COMMAND}\""
    exit 1
    # "scripts/pdsadmin/${COMMAND}.sh" "$@"
    ;;
esac
