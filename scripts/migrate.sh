#!/bin/bash

set -e

DIR=db/migrations/api
DB=""
ENV_FILES=()

function help() {
  echo "$1 [-h|--help|-env|-docker] -- [create] <args...>"
  echo "  -env        environment file (default: .env)"
  echo "  -d          use a different database"
  echo "  -docker     use docker to run the command"
  echo "  -h, --help  print help message"
}

while [ $# -gt 0 ]; do
  case $1 in
    -h|--help)
      help "migrate.sh"
      exit
      ;;
    -d|--database)
    	DB="$2"
      shift 2
      ;;
    -env)
      ENV_FILES+=("$2")
      shift 2
      ;;
    --)
      shift
      break
      ;;
    *)
      break
      ;;
    esac
done

if [ ${#ENV_FILES} -eq 0 ]; then
  ENV_FILES+=("config/env/db.env")
fi

for file in "${ENV_FILES[@]}"; do
  if [ ! -f "$file" ]; then
    echo "Error: $file does not exist"
    exit 1
  fi
  source "$file"
done

if [ -z "$DATABASE_URL" ]; then
  DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${DB:-$POSTGRES_DB}?sslmode=disable"
fi

unset PGSERVICEFILE

case $1 in
  create)
    run-migrate create -ext sql -seq -dir "$DIR" "$2"
    ;;
  *)
    migrate -source "file://$DIR" -database "$DATABASE_URL" "$@"
    ;;
esac

