#!/bin/bash

set -e

ENV_FILE=.env
DIR=db/migrations

while :; do
  case $1 in
    -h|--help)
      echo "migrate.sh [-h|-help|-env] [create] <args...>"
      exit
      ;;
    -env)
      ENV_FILE="$2"
      shift 2
      ;;
    *)
      break
      ;;
    esac
done

if [ ! -f "$ENV_FILE" ]; then
  echo "$ENV_FILE does not exist"
  exit 1
fi

source "$ENV_FILE"

unset PGSERVICEFILE

case $1 in
  create)
    migrate create -ext sql -seq -dir "$DIR" $2
  ;;
  *)
    migrate -source "file://$DIR" -database "$DATABASE_URL" $@
  ;;
esac
