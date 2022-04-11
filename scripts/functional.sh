#!/bin/sh

set -e

help() {
    echo "$1 [build|setup|run|stop]"
}

case $1 in
  -h|--help)
    help "$0"
    exit
    ;;
  build)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml build
    ;;
  setup)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d db redis web
    ;;
  run|test)
    shift
    docker-compose -f docker-compose.yml -f docker-compose.test.yml run -u "$(id -u):$(id -g)" --rm tests scripts/functional-tests.sh "$@"
    ;;
  stop)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml down
    ;;
  *)
    help "$0"
    echo "Error: unknown command"
    exit 1
    ;;
esac

