#!/bin/bash

set -e

ENV_FILE=.env
USE_DOCKER=false

while :; do
  case $1 in
    -h|--help)
      echo "psql.sh [-h|-help|-env|-docker] <psql args...>"
      exit
      ;;
    -env)
      ENV_FILE="$2"
      shift 2
      ;;
    -docker)
      USE_DOCKER=true
      shift
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

if $USE_DOCKER; then
  # Reset the port because we are running this in a docker container
  echo 'using docker-compose'
  POSTGRES_PORT=5432
  PGPASS_LINE="$POSTGRES_HOST:$POSTGRES_PORT:$POSTGRES_DB:$POSTGRES_USER:$POSTGRES_PASSWORD"
  docker-compose exec -e PGPASS_LINE="$PGPASS_LINE" db bash -c 'echo $PGPASS_LINE > /root/.pgpass && chmod 0600 /root/.pgpass'
  docker-compose exec             \
    -e PGPASSFILE='/root/.pgpass' \
    -e PSQL_PAGER=less            \
    db psql             \
    -h "$POSTGRES_HOST" \
    -p "$POSTGRES_PORT" \
    "$POSTGRES_DB" "$POSTGRES_USER" "$@"
else
  if [ ! -d config/postgres ]; then mkdir -p config/postgres; fi
  PGPASSFILE="config/postgres/$(basename $ENV_FILE).pgpass"
  PGPASS_LINE="$POSTGRES_HOST:$POSTGRES_PORT:$POSTGRES_DB:$POSTGRES_USER:$POSTGRES_PASSWORD"
  echo "$PGPASS_LINE" > $PGPASSFILE
  chmod 0600 $PGPASSFILE
  PGPASSFILE="$PGPASSFILE" psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" "$POSTGRES_DB" "$POSTGRES_USER" $@
fi

# vim: ts=2 sts=2 sw=2
