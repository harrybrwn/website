#!/usr/bin/env bash

set -euo pipefail

usage() {
	echo "Usage"
	echo "  from-env-file.sh <filename> [flags]"
	echo
	echo "Flags"
	echo "  -env <stg|prd>        use an environment"
	echo "  -name <name>          change the name of the secret"
	echo "  -n -namespace <val>   change the namespace of the secret (this cannot be changed later on)"
	echo "  -dry-run              Don't write the secret do disk"
}

protect_cluster_by_env() {
	local env="$1"
	local count="$(kubectl get nodes -l "hrry.me/env=$env" -o json | jq -r '.items | length')"
	if [ ${count} -eq 0 ]; then
		echo "[error]: currently not looking at cluster with 'hrry.me/env=$env'"
		exit 1
	fi
}

create() {
	local FILE="$1"
	local NAME="$2"
	shift 2
	kubectl create --dry-run=client secret generic \
		--from-env-file "${FILE}" \
		"${NAME}" \
		--output yaml \
		"$@"
}

readonly CONTROLLER_NAME="sealed-secrets"
readonly CONTROLLER_NAMESPACE="secrets-management"

kseal() {
	kubeseal \
		--controller-namespace "$CONTROLLER_NAMESPACE" \
		--controller-name "$CONTROLLER_NAME" \
		--format yaml \
		"$@"
}

# Script flags
ENV=""
FILE=""
NAME=""
DRY=false
# OUT=/dev/stdout

# kubesesal flags
KS_NAMESPACE="default"

while [ $# -gt 0 ]; do
	case "$1" in
		-env|--env)
			ENV="$2"
			shift 2
			;;
		-name|--name)
			NAME="$2"
			shift 2
			;;
		-dry-run|--dry-run)
		 	DRY=true
			shift
			;;
		-o|-out|--output)
			# OUT="$2"
			shift 2
			;;
		-n|-namespace|--namespace)
			KS_NAMESPACE="$2"
			shift 2
			;;
		-h|-help|--help)
			usage
			exit 0
			;;
		*)
			if [ -z "${FILE}" ]; then
				FILE="$1"
				shift
			else
				echo "[error]: unknown flag \"$1\""
				exit 1
			fi
	esac
done

# Validate ENV
case "$ENV" in
	stg|prd)
		;;
	*)
		echo "Environment \"$ENV\" is not supported"
		exit 1
		;;
esac
protect_cluster_by_env "${ENV}"

if [ -z "${FILE}" ]; then
	echo "[error]: <filename> is a required argument"
	echo
	usage
	exit 1
elif [ ! -f "${FILE}" ]; then
	echo "[error]: file \"${FILE}\" does not exist"
	exit 1
fi

if [ -z "${NAME}" ]; then
	NAME="$(basename "${FILE}" .env)"
fi

if [ -f "${ENV}/${NAME}.yml" ]; then
	echo "[error]: sealed secret \"${ENV}/${NAME}.yml\" already exists, delete it to move forward"
	exit 1
fi

if $DRY; then
	create "${FILE}" "${NAME}" --namespace "${KS_NAMESPACE}"
	echo "---"
	create "${FILE}" "${NAME}" --namespace | kseal --namespace "${KS_NAMESPACE}"
else
	create "${FILE}" "${NAME}" --namespace "${KS_NAMESPACE}" \
	| kseal \
			--namespace "${KS_NAMESPACE}" \
			--sealed-secret-file "${ENV}/${NAME}.yml"
fi
