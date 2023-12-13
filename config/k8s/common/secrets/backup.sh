#!/usr/bin/env bash

set -euo pipefail

usage() {
	echo "Usage"
	echo "  backup.sh [flags]"
	echo
	echo "Flags"
	echo "  -env <stg|prd>  cluster environment"
	echo "  -h, -help       print usage"
	echo "  -dry-run        evaluate but don't write to disk"
}

protect_cluster_by_env() {
	local env="$1"
	local count="$(kubectl get nodes -l "hrry.me/env=$env" -o json | jq -r '.items | length')"
	if [ ${count} -eq 0 ]; then
		echo "[error]: currently not looking at cluster with 'hrry.me/env=$env'"
		echo
		echo "Check with: kubectl get nodes -l 'hrry.me/env=$env'"
		exit 1
	fi
}

readonly CONTROLLER_NAMESPACE="secrets-management"

# Script flags
ENV=""
DRY=false
while [ $# -gt 0 ]; do
	case "$1" in
		-env|--env)
			ENV="$2"
			shift 2
			;;
		-s3|--s3)
			echo "[error]: S3 backup not supported yet"
			exit 1
			;;
		-h|-help|--help)
			usage
			exit 0
			;;
		-dry|--dry|-dry-run|--dry-run)
		 	DRY=true
			shift
			;;
		*)
			echo "[error]: unknown flag \"$1\""
			exit 1
	esac
done

# Validate ENV
case "$ENV" in
	stg|prd)
		;;
	*)
		echo "Environment \"$ENV\" is not supported: use -env <stg|prd>"
		exit 1
		;;
esac
protect_cluster_by_env "${ENV}"

if [ ! -d "${ENV}" ]; then
	mkdir "${ENV}"
fi

readonly current_key_yaml="$(kubectl get secret                    \
	-n "${CONTROLLER_NAMESPACE}"                                   \
	--selector sealedsecrets.bitnami.com/sealed-secrets-key=active \
    --field-selector 'type=kubernetes.io/tls'                      \
    --sort-by '{.metadata.creationTimestamp}'                      \
	-o yaml)"
# readonly current_key_json="$(kubectl get secret                    \
# 	-n "${CONTROLLER_NAMESPACE}"                                   \
# 	--selector sealedsecrets.bitnami.com/sealed-secrets-key=active \
#     --sort-by '{.metadata.creationTimestamp}'                      \
# 	-o 'jsonpath={.items[-1]}')"

readonly nkeys="$(echo -n "$current_key_yaml" | yq -r '.items | length')"
if [ ${nkeys} -eq 0 ]; then
	echo "[error]: no seal-secrets keys found"
	exit 1
fi

readonly date="$(date '+%FT%T%z')"
readonly BACKUP_FILE="${ENV}/seal-key.backup.${date}.yml"

readonly cur_key="$(echo "${current_key_yaml}" | yq -r '.items[-1].data."tls.key"')"
readonly cur_crt="$(echo "${current_key_yaml}" | yq -r '.items[-1].data."tls.crt"')"
if [ -f "${BACKUP_FILE}" ]; then
	old_key="$(cat "${BACKUP_FILE}" | yq -r '.items[-1].data."tls.key"')"
	old_crt="$(cat "${BACKUP_FILE}" | yq -r '.items[-1].data."tls.crt"')"
else
	old_key=''
	old_crt=''
fi

if [ "${cur_key}" = "${old_key}" ]; then
	echo "Current private key has not changed, no backup needed."
	exit 0
elif [ "${cur_crt}" = "${old_crt}" ]; then
	echo "Current certificate has not changed, no backup needed."
	exit 0
else
	echo "Writing seal key to disk"
	if $DRY; then
      echo "Writing to:"
      echo "  ${BACKUP_FILE}"
      echo "  ${ENV}/seal.${date}.key"
      echo "  ${ENV}/seal.${date}.crt"
      echo "Dry run finished."
      exit 0
	fi
	echo "${current_key_yaml}" > "${BACKUP_FILE}"
	echo -n "${current_key_yaml}" | yq -r '.items[-1].data."tls.key"' | base64 -d > "${ENV}/seal.${date}.key"
	echo -n "${current_key_yaml}" | yq -r '.items[-1].data."tls.crt"' | base64 -d > "${ENV}/seal.${date}.crt"
fi
