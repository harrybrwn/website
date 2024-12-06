#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

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

SUBCOMMAND="${1:-}"

if [ -z "${PDS_HOSTNAME}" ]; then
  echo "Error: PDS_HOSTNAME is required" 1>&2
  exit 1
elif [ -z "${PDS_ADMIN_PASSWORD}" ]; then
  echo "Error: PDS_ADMIN_PASSWORD is required" 1>&2
  exit 1
fi

# set -- "${ARGS[@]}"

case "${SUBCOMMAND}" in
  #
  # account list
  #
  list)
    DIDS="$(curl_cmd_get \
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.sync.listRepos?limit=100" | jq --raw-output '.repos[].did'
    )"
    OUTPUT='[{"handle":"Handle","email":"Email","did":"DID"}'
    for did in ${DIDS}; do
      ITEM="$(curl_cmd_get \
        --user "admin:${PDS_ADMIN_PASSWORD}" \
        "https://${PDS_HOSTNAME}/xrpc/com.atproto.admin.getAccountInfo?did=${did}"
      )"
      OUTPUT="${OUTPUT},${ITEM}"
    done
    OUTPUT="${OUTPUT}]"
    echo "${OUTPUT}" | jq --raw-output '.[] | [.handle, .email, .did] | @tsv' | column --table
    ;;

  #
  # account list-dids
  #
  list-dids)
    DIDS="$(curl_cmd_get \
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.sync.listRepos?limit=100" | jq --raw-output '.repos[].did'
    )"
    for did in ${DIDS}; do
      echo "$did"
    done
    ;;

  #
  # account create
  #
  create)
    EMAIL="${2:-}"
    HANDLE="${3:-}"

    if [[ "${EMAIL}" == "" ]]; then
      read -p "Enter an email address (e.g. alice@${PDS_HOSTNAME}): " EMAIL
    fi
    if [[ "${HANDLE}" == "" ]]; then
      read -p "Enter a handle (e.g. alice.${PDS_HOSTNAME}): " HANDLE
    fi

    if [[ "${EMAIL}" == "" || "${HANDLE}" == "" ]]; then
      echo "ERROR: missing EMAIL and/or HANDLE parameters." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <EMAIL> <HANDLE>" >/dev/stderr
      exit 1
    fi

    PASSWORD="$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-24)"
    INVITE_CODE="$(curl_cmd_post \
      --user "admin:${PDS_ADMIN_PASSWORD}" \
      --data '{"useCount": 1}' \
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.server.createInviteCode" | jq --raw-output '.code'
    )"
    RESULT="$(curl_cmd_post_nofail \
      --data "{\"email\":\"${EMAIL}\", \"handle\":\"${HANDLE}\", \"password\":\"${PASSWORD}\", \"inviteCode\":\"${INVITE_CODE}\"}" \
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.server.createAccount"
    )"

    DID="$(echo $RESULT | jq --raw-output '.did')"
    if [[ "${DID}" != did:* ]]; then
      ERR="$(echo ${RESULT} | jq --raw-output '.message')"
      echo "ERROR: ${ERR}" >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <EMAIL> <HANDLE>" >/dev/stderr
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
    ;;

  #
  # account delete
  #
  delete)
    DID="${2:-}"

    if [[ "${DID}" == "" ]]; then
      echo "ERROR: missing DID parameter." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
      exit 1
    fi

    if [[ "${DID}" != did:* ]]; then
      echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
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
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.admin.deleteAccount" >/dev/null

    echo "${DID} deleted"
    ;;

  #
  # account takedown
  #
  takedown)
    DID="${2:-}"
    TAKEDOWN_REF="$(date +%s)"

    if [[ "${DID}" == "" ]]; then
      echo "ERROR: missing DID parameter." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
      exit 1
    fi

    if [[ "${DID}" != did:* ]]; then
      echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
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
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateSubjectStatus" >/dev/null

    echo "${DID} taken down"
    ;;

  #
  # account untakedown
  #
  untakedown)
    DID="${2:-}"

    if [[ "${DID}" == "" ]]; then
      echo "ERROR: missing DID parameter." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
      exit 1
    fi

    if [[ "${DID}" != did:* ]]; then
      echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
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
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateSubjectStatus" >/dev/null

    echo "${DID} untaken down"
    ;;

  #
  # account reset-password
  #
  reset-password)
    DID="${2:-}"
    PASSWORD="$(openssl rand -base64 30 | tr -d "=+/" | cut -c1-24)"

    if [[ "${DID}" == "" ]]; then
      echo "ERROR: missing DID parameter." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
      exit 1
    fi

    if [[ "${DID}" != did:* ]]; then
      echo "ERROR: DID parameter must start with \"did:\"." >/dev/stderr
      echo "Usage: $0 ${SUBCOMMAND} <DID>" >/dev/stderr
      exit 1
    fi

    curl_cmd_post \
      --user "admin:${PDS_ADMIN_PASSWORD}" \
      --data "{ \"did\": \"${DID}\", \"password\": \"${PASSWORD}\" }" \
      "https://${PDS_HOSTNAME}/xrpc/com.atproto.admin.updateAccountPassword" >/dev/null

    echo
    echo "Password reset for ${DID}"
    echo "New password: ${PASSWORD}"
    echo
    ;;

  test)
    echo "this is a test"
    echo "testing testing 123..."
    ;;

  *)
    echo "Unknown subcommand: \"${SUBCOMMAND}\"" 1>&2
    exit 1
    ;;
esac
