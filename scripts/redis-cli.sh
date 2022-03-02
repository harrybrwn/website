#!/bin/bash

set -e

ENV_FILE=.env

function help() {
  echo "$1 [-h|--help|-env] -- <args...>"
  echo "  -env        environment file (default: .env)"
  echo "  -h, --help  print help message"
}

while :; do
  case $1 in
    -h|--help)
      help "redis-cli.sh"
      exit
      ;;
    -env)
      ENV_FILE="$2"
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

if [ ! -f "$ENV_FILE" ]; then
  echo "$ENV_FILE does not exist"
  exit 1
fi

source "$ENV_FILE"

docker-compose exec                  \
  -e REDISCLI_AUTH="$REDIS_PASSWORD" \
  redis redis-cli  \
  -h "$REDIS_HOST" \
  -p "$REDIS_PORT" $@

# vim: ts=2 sts=2 sw=2
