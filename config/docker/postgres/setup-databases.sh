#!/bin/bash

set -eu

function create_user_and_database() {
	local database=$(printf "%s" "$1" | cut -d: -f1)
	local owner=$(printf "%s" "$1" | cut -d: -f2 -s)
	if [ -z "${database:-}" ]; then
		echo "Error: no database name" 1>&2
		exit 1
	fi
	if [ -n "${owner:-}" ]; then
		echo "Creating database \"$database\" with owner \"$owner\""
		psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
				CREATE DATABASE "$database" WITH OWNER "$owner";
		EOSQL
	else
		echo "Creating database \"$database\""
		psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
				CREATE DATABASE "$database";
		EOSQL
	fi
}

function create_user() {
	local user=$(printf "%s" "$1" | cut -d: -f1)
	local pw=$(printf "%s" "$1" | cut -d: -f2)
	if [ -z "${user:-}" -o -z "${pw:-}" ]; then
		echo "Error: must have username and password" 1>&2
		exit 1
	fi
	echo "Creating new user \"$user\""
	psql -v ON_ERROR_STOP=1 --username "${POSTGRES_USER}" <<-EOSQL
		CREATE ROLE "$user" WITH LOGIN PASSWORD '$pw';
	EOSQL
}

if [ -n "${POSTGRES_USERS:-}" ]; then
	for pair in ${POSTGRES_USERS}; do
		create_user "$pair"
	done
fi

if [ -n "${POSTGRES_DATABASES:-}" ]; then
	echo "Multiple database creation requested: $POSTGRES_DATABASES"
	for db in $(echo $POSTGRES_DATABASES | tr ',' ' '); do
		create_user_and_database $db
	done
	echo "Multiple databases created"
fi