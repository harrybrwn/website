#!/usr/bin/env bash

set -euo pipefail

query_key() {
	yq -r '.items[0].data."tls.key"' "$@"
}

query_crt() {
	yq -r '.items[0].data."tls.crt"' "$@"
}

readonly BACKUP_FILE=secrets-backup.yml
ENV='dev'

while [ $# -gt 0 ]; do
	case "$1" in
		-env|--env)
			ENV="$2"
			shift 2
			;;
		-s3|--s3)
			echo "Error: S3 backup not supported yet"
			exit 1
			;;
		*)
			echo "Error: unknown flag \"$1\""
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

# Go to secrets dir
if [ -d "config/k8s/${ENV}/secrets" ]; then
	pushd "config/k8s/${ENV}/secrets"
elif [ -d "config/k8s/${ENV}" ]; then
	mkdir "config/k8s/${ENV}/secrets"
	pushd "config/k8s/${ENV}/secrets"
else
	pushd .
fi
trap popd EXIT

current_key_yaml="$(kubectl get secret \
	-n secrets-management \
	-l sealedsecrets.bitnami.com/sealed-secrets-key \
	-o yaml)"

if [ -f "$BACKUP_FILE" ]; then
	old_key="$(query_key "$BACKUP_FILE")"
	old_crt="$(query_crt "$BACKUP_FILE")"
	current_key="$(echo "$current_key_yaml" | query_key)"
	current_crt="$(echo "$current_key_yaml" | query_crt)"
	if [ "$current_key" = "$old_key" ] && [ "$current_crt" = "$old_crt" ]; then
		echo backup key is unchanged
	else
		echo "Writing keys to $BACKUP_FILE"
		echo "$current_key_yaml" > "$BACKUP_FILE"
	fi
else
	echo "Writing keys to $BACKUP_FILE"
	echo "$current_key_yaml" > "$BACKUP_FILE"
fi
