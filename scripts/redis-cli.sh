#!/bin/bash

set -e

ENV_FILE=.env

while :; do
  case $1 in
    -h|--help)
      echo "redis-cli.sh [-h|-help|-env]"
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

docker-compose exec                  \
  -e REDISCLI_AUTH="$REDIS_PASSWORD" \
  redis redis-cli  \
  -h "$REDIS_HOST" \
  -p "$REDIS_PORT" $@

# vim: ts=2 sts=2 sw=2