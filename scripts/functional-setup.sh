#!/bin/sh

set -e

scripts/wait.sh "$POSTGRES_HOST" "$POSTGRES_PORT" -w 1 -- scripts/migrate.sh -env none -- up

