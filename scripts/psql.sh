#!/bin/bash

set -e

DOCKER=false
ENV_FILES=()
DB=""

function help() {
  echo "$1 [-h|--help|-env|-docker] -- <args...>"
  echo "  -env        environment file (default: .env)"
  echo "  -docker     use docker to run the command"
  echo "  -h, --help  print help message"
}

while [ $# -gt 0 ]; do
  case $1 in
    -h|--help)
      help "psql.sh"
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

if $DOCKER; then
  # Reset the port because we are running this in a docker container
  echo 'using docker-compose'
  POSTGRES_PORT=5432
  uri="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${DB:-$POSTGRES_DB}"
  docker-compose exec -e PSQL_PAGER=less db psql "${uri}" "$@"
else
  uri="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-localhost}:${POSTGRES_PORT:-5432}/${DB:-$POSTGRES_DB}"
  psql "${uri}" "$@"
fi

# vim: ts=2 sts=2 sw=2
