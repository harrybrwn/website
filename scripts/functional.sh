#!/bin/bash

set -euo pipefail

# readonly DIR="$(pwd)/$(dirname "${BASH_SOURCE[0]}")"
readonly SCRIPT="$0"

get-help() {
  case "$1" in
    run|test)
      echo "Run functional test suite containers."
      echo
      echo "Usage"
      echo "  $SCRIPT <test|run> [...pytest options] [...test files]"
      echo
      echo "Pytest Options"
      echo "  -k <name>  run a test by name"
      echo
      ;;
    stop|down)
      echo "Stop all containers."
      echo
      echo "Usage"
      echo "  $SCRIPT <stop|down>"
      echo
      ;;
    help)
      echo "Get help message for any sub command if it has a help page."
      echo
      echo "Usage"
      echo "  $SCRIPT help [command]"
      echo
      ;;
    *)
      return 1
      ;;
  esac
  return 0
}

function help() {
  local ret=0
  if [ -n "${1:-}" ]; then
    if get-help "$@"; then
      return 0
    else
      echo "Error: no help page for command \"$1\""
      echo
      ret=1
    fi
  fi
  echo "Manage functional tests and associated containers."
  echo
  echo "Usage"
  echo "  $SCRIPT [command] [...options]"
  echo
  echo "Commands"
  echo "  build   build images"
  echo "  setup   setup depenant containers"
  echo "  test    run the test container and all tests"
  echo "  stop    tear down all running containers"
  echo "  ps      list running containers"
  echo "  logs    view container logs"
  echo "  help    get help on a sub-command if it has a help page"
  echo
  echo "Options"
  echo "  -h --help   Print this help message"
  echo
  return $ret
}

readonly SERVICES=("db" "redis" "nginx" "api" "hooks" "legacy-site" "geoip")

#############
# Utilities #
#############

compose() {
  docker compose -f docker-compose.yml -f config/docker-compose.test.yml "$@"
}

running() {
  for s in "${SERVICES[@]}"; do
    local id running
    id="$(compose ps --quiet "$s")"
    running="$(docker container inspect "${id}" | jq -r '.[0].State.Running')"
    if [ "${running}" != "true" ]; then
      echo "service $s is down"
      return 1
    fi
  done
  return 0
}

############
# Commands #
############

declare -x GIT_COMMIT SOURCE_HASH
GIT_COMMIT=$(git rev-parse HEAD)
SOURCE_HASH=$(./scripts/sourcehash.sh -e '*_test.go')

build() {
   compose build "${SERVICES[@]}" tests
}

setup() {
  compose up -d "${SERVICES[@]}"
}

run_tests() {
  local pytest_args script
  pytest_args="${@:-test/}"
  script=$(cat <<-EOF
scripts/wait.sh "\${POSTGRES_HOST}:\${POSTGRES_PORT}" -w 1 -- migrate.sh up
scripts/wait.sh "\${APP_HOST}:\${APP_PORT:-443}" -w 1 -- pytest -s ${pytest_args}
EOF
)
  compose run \
    -u "$(id -u):$(id -g)" --rm tests bash -c "$script"
}

stop() {
  compose down "$@"
}

ps() {
  compose ps "$@"
}

logs() {
  compose logs "$@"
}

main() {
  CMD=""
  ARGS=()
  # COLLECT_ALL is set to true when passing -- as an argument. This is used to
  # pass flags to programs beeing run in sub-commands.
  COLLECT_ALL=false

  while [ $# -gt 0 ]; do
    case $1 in
      --)
        COLLECT_ALL=true
        shift
        ;;
      -h|--help)
        if $COLLECT_ALL; then
          ARGS+=("$1")
          shift
        else
          help
          exit
        fi
        ;;
      -*)
        if [ -z "$CMD" ] && ! $COLLECT_ALL; then
          echo "Error: unknown flag \"$1\""
          help
          exit 1
        else
          ARGS+=("$1")
          shift
        fi
        ;;
      *)
        if [ -z "$CMD" ]; then
          CMD="$1"
        else
          ARGS+=("$1")
        fi
        shift
        ;;
    esac
  done

  case $CMD in
    help)
      help "${ARGS[@]}"
      exit
      ;;
    build)
      "$CMD" "${ARGS[@]}"
      ;;
    setup)
      "$CMD" "${ARGS[@]}"
      ;;
    run|test)
      run_tests "${ARGS[@]}"
      ;;
    stop|down)
      stop "${ARGS[@]}"
      ;;
    ps)
      "$CMD" "${ARGS[@]}"
      ;;
    logs)
      "$CMD" "${ARGS[@]}"
      ;;
    *)
      help "${ARGS[@]}"
      if [ -n "$CMD" ]; then
        echo "Error: unknown command \"$CMD\""
        exit 1
      fi
      exit
      ;;
  esac
}

main "$@"
