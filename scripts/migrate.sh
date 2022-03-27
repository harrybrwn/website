#!/bin/bash

set -e

ENV_FILE=.env
DIR=db/migrations
DOCKER=false

function help() {
  echo "$1 [-h|--help|-env|-docker] -- [create] <args...>"
  echo "  -env        environment file (default: .env)"
  echo "  -docker     use docker to run the command"
  echo "  -h, --help  print help message"
}

while :; do
  case $1 in
    -h|--help)
      help "migrate.sh"
      exit
      ;;
    -env)
      ENV_FILE="$2"
      shift 2
      ;;
    -docker)
      DOCKER=true
      shift
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

# Don't fail with no env file. Will use this script in ci and containers.
if [ -f "$ENV_FILE" ]; then
  source "$ENV_FILE"
fi

if [ -z "$DATABASE_URL" ]; then
  DATABASE_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${POSTGRES_PORT}/${POSTGRES_DB}?sslmode=disable"
fi

unset PGSERVICEFILE

run-migrate() {
  if $DOCKER; then
    docker container run \
      --rm               \
      --network host     \
      -v "$(pwd)/$DIR:/migrations" -it migrate/migrate:latest
  else
    migrate "$@"
  fi
}

case $1 in
  create)
    run-migrate create -ext sql -seq -dir "$DIR" $2
    ;;
  *)
    run-migrate -source "file://$DIR" -database "$DATABASE_URL" "$@"
    ;;
esac
