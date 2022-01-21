#!/bin/bash

set -e

# psql -h localhost -p 9432 harrybrwn harrybrwndev "$@"

ENV_FILE=.env

while :; do
  case $1 in
    -h|--help)
      echo "psql.sh [-h|-help|-env]"
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
  echo '.env does not exist'
  exit 1
fi

source "$ENV_FILE"

# Reset the port because we are running this in a docker container
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
