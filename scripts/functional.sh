#!/bin/sh

set -e

case $1 in
  -h|--help)
    echo "$0 [build|setup|run|stop]"
    ;;
  build)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml build
    ;;
  setup)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d db redis web
    ;;
  run)
    shift
    docker-compose -f docker-compose.yml -f docker-compose.test.yml run -u "$(id -u):$(id -g)" --rm tests scripts/functional-tests.sh "$@"
    ;;
  stop)
    docker-compose -f docker-compose.yml -f docker-compose.test.yml down
    ;;
  *)
    echo "Error: unknown command"
    exit 1
    ;;
esac
