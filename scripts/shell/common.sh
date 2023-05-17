#!/bin/bash

declare -r CERTDB="sql:$HOME/.pki/nssdb"
declare -r LOCAL_CERT_NAME="harrybrwn-local-dev"
export CERTDB LOCAL_CERT_NAME

readonly RED="\e[31m"
readonly GREEN="\e[32m"
readonly CYAN="\e[36m"
readonly YELLOW="\e[1;33m"
readonly NOCOL="\e[0m"
export RED GREEN CYAN YELLOW NOCOL

in_docker() {
	grep -q docker /proc/1/cgroup || [ -f /.dockerenv ]
}

has_certutil() {
	command -v certutil > /dev/null 2>&1
}

info() {
	_log="$1"
	shift
	echo -e "${GREEN}[$_log]${NOCOL} $*"
	unset _log
}

warn() {
	_log="$1"
	shift
	echo -e "${YELLOW}[$_log]${NOCOL} $*"
	unset _log
}

error() {
	_log="$1"
	shift
	echo -e "${RED}[$_log]${NOCOL} $*" 1>&2
	unset _log
}

list-images() {
	blob="$(docker buildx bake \
		--file config/docker/docker-bake.hcl \
		--print 2>&1 \
		| jq -r '.target[] | .tags[]' \
		| sort \
		| uniq \
		| grep -E 'latest$')"

	if [ -n "${*:-}" ]; then
		img=''
		match="${*}"
		for e in $blob; do
			if [[ "$e" =~ $match ]]; then
				if [ -n "$img" ]; then
					img="$img
$e"
				else
					img="$e"
				fi
			fi
		done
		if [ -z "${img}" ]; then
			return 1
		fi
		blob="$img"
	fi
	printf '%s' "$blob"
}

k3d_cluster_name() {
	_name="$(yq -r '.metadata.name' "${1}")"
	if [ -z "${_name}" ]; then
		unset _name
	 	error util "Failed to find k3d cluster name"
		return 1
	fi
	printf '%s' "${_name}"
	unset _name
}

k3d_running() {
  if [ "$(k3d cluster get "${1}" -o json 2>/dev/null | jq -M '.[0].serversRunning')" = "1" ]; then
    return 0
  else
    return 1
  fi
}
