#!/bin/sh

set -eu

DEBUG=false
log() {
  if $DEBUG; then
    echo "$*"
  fi
}

error() {
  printf "Error: %s\n" "$*" 1>&2
}

WAIT=1
TIMEOUT=15
HOSTS=""
ARGS=""

help() {
  echo "Wait for a remote endpoint to be accessable before"
  echo "continuing or running a command."
  echo
  echo "Usage"
  echo "  wait.sh <locations...> [flags...] [--] <command>"
  echo
  echo "Flags"
  echo "     --help     print help message"
  echo "  -w --wait     seconds to wait between failures (default: $WAIT)"
  echo "  -t --timeout  timeout in seconds (default: $TIMEOUT)"
  echo
  echo "Examples"
  echo '  $ wait.sh http://localhost:8080 -- ls'
  echo '  $ wait.sh tcp://example.com:80'
  echo '  $ wait.sh -t 2 localhost:4444 -- echo yes'
  echo '  $ wait.sh udp://localhost:5432 --wait 10 -- echo up...'
}

while [ $# -gt 0 ]; do
  case $1 in
    http://*|https://*|tcp://*|udp://*|*:*)
      if [ -z "${HOSTS}" ]; then
        HOSTS="${1}"
      else
        HOSTS="${HOSTS} ${1}"
      fi
      shift 1
      ;;
    --help|-h)
      help
      exit
      ;;
    -w|--wait)
      WAIT=$2
      shift 2
      ;;
    -t|--timeout)
      TIMEOUT=$2
      shift 2
      ;;
    --)
      shift
      ARGS="$*"
      break
      ;;
    *)
      error "unknown argument $1"
      help
      exit 1
      ;;
    esac
done

set -- "$HOSTS"
TIMEOUT_END=$(($(date +%s) + TIMEOUT))

while [ $# -gt 0 ]; do
  CMD=""
  endpoint="$1"
  case "$1" in
    http://*|https://*)
    	HOST="$1"
      if [ -z "${HOST}" ]; then
        error "empty host"
        exit 1
      fi
      CMD="wget --timeout=1 -q -O- ${HOST}"
      shift 1
      ;;
    udp://*)
      endpoint=$(printf "%s\n" "$1" | sed -e 's/udp:\/\///g')
      HOST=$(printf "%s\n" "${endpoint}"| cut -d : -f 1)
      PORT=$(printf "%s\n" "${endpoint}"| cut -d : -f 2)
      if [ -z "${HOST}" ]; then
        error "empty host"
        exit 1
      elif [ -z "${PORT}" ]; then
        error "empty port"
        exit 1
      fi
      CMD="nc -u -w 1 -z ${HOST} ${PORT}"
      shift 1
      ;;
    *:*)
      endpoint=$(printf "%s\n" "$1" | sed -e 's/tcp:\/\///g')
      HOST=$(printf "%s\n" "${endpoint}"| cut -d : -f 1)
      PORT=$(printf "%s\n" "${endpoint}"| cut -d : -f 2)
      if [ -z "${HOST}" ]; then
        error "empty host"
        exit 1
      elif [ -z "${PORT}" ]; then
        error "empty port"
        exit 1
      fi
      CMD="nc -w 1 -z ${HOST} ${PORT}"
      shift 1
      ;;
    *)
      error "unknown protocol for host '$1'"
      exit 1
      ;;
  esac

  while :; do
    set +e
    $CMD > /dev/null 2>&1
    result=$?
    set -e
    log "'$CMD' => $result"
    if [ $result -eq 0 ]; then
      break
    fi

    log "sleeping for $WAIT..."
    sleep "$WAIT"
    if [ "$TIMEOUT" -ne 0 ] && [ "$(date +%s)" -ge "$TIMEOUT_END" ]; then
      error "timeout"
      exit 2
    fi
  done
done

if [ -n "$ARGS" ]; then
  # shellcheck disable=SC2086
  exec $ARGS
fi