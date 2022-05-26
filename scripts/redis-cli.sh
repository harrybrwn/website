#!/bin/bash

set -e

ENV_FILES=()

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
  ENV_FILES+=(".env" "config/env/redis.env")
fi

for file in "${ENV_FILES[@]}"; do
  if [ ! -f "$file" ]; then
    echo "Error: $file does not exist"
    exit 1
  fi
  source "$file"
done

docker-compose exec                  \
  -e REDISCLI_AUTH="$REDIS_PASSWORD" \
  redis redis-cli  \
  -h "${REDIS_HOST:-localhost}" \
  -p "${REDIS_PORT:-6379}" "$@"

# vim: ts=2 sts=2 sw=2
