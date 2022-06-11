#!/bin/sh

# set -e

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
HOST=""
PORT=""
HOSTS=""
ARGS=""

help() {
  echo "wait.sh <host> <port> [flags...] [--] <command>"
  echo "Flags:"
  echo "     --help     print help message"
  echo "  -h --host     host of service to wait for"
  echo "  -p --port     port of service to wait for"
  echo "  -w --wait     seconds to wait between failures (default $WAIT)"
  echo "  -t --timeout  timeout in seconds"
}

waitfor() {
  case $1 in
    http://*|https://*)
      ;;
    *:*)
      ;;
  esac
}

while [ $# -gt 0 ]; do
  case $1 in
    http://*|https://*|*:*)
      if [ -z "${HOSTS}" ]; then
        HOSTS="${1}"
      else
        HOSTS="${HOSTS} ${1}"
      fi
      shift 1
      ;;
    # *:*)
    #   if [ -z "${HOSTS}" ]; then
    #     HOSTS="${1}"
    #   else
    #     HOSTS="${HOSTS} ${1}"
    #   fi
    #   shift 1
    #   ;;
    --help)
      help
      exit
      ;;
    -h|--host)
      HOST="$2"
      shift 2
      ;;
    -p|--port)
      PORT="$2"
      shift 2
      ;;
    -w|--wait)
      WAIT=$2
      shift 2
      ;;
    -t|--timeout)
      TIMEOUT="1"
      exit 2
      ;;
    --)
      shift
      ARGS="$@"
      break
      ;;
    *)
      if [ -z "$1" ]; then
        break
      elif [ -z "$HOST" ]; then
        HOST="$1"
      elif [ -z "$PORT" ]; then
        PORT="$1"
      fi
      shift
      ;;
    esac
done

set -- $HOSTS
OK=false
TIMEOUT_END=$(($(date +%s) + $TIMEOUT))

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
    *:*)
      HOST=$(printf "%s\n" "$1"| cut -d : -f 1)
      PORT=$(printf "%s\n" "$1"| cut -d : -f 2)
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
    $CMD > /dev/null 2>&1
    result=$?
    log "'$CMD' => $result"
    if [ $result -eq 0 ]; then
      break
    fi

    log "sleeping for $WAIT..."
    sleep $WAIT
    if [ $TIMEOUT -ne 0 -a $(date +%s) -ge $TIMEOUT_END ]; then
      error "timeout"
      exit 2
    fi
  done
done

log "args: $ARGS"
if [ -n "${ARGS}" ]; then
  exec "$ARGS"
fi