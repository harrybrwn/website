#!/bin/sh

set -e

HOST=
PORT=
WAIT=1

help() {
  echo "$1 [flags...] [--] <command>"
  echo "Flags:"
  echo "     --help   print help message"
  echo "  -h --host   host of service to wait for"
  echo "  -p --port   port of service to wait for"
  echo "  -w --wait   wait time between lookup failures (seconds)"
}

while :; do
  case $1 in
    --help)
      help "wait.sh"
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
      echo "Error: timeout not supported yet"
      exit 1
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

if [ -z "$HOST" ]; then
  echo "Error: no host"
  exit 1
fi
if [ -z "$PORT" ]; then
  echo "Error: no port"
  exit 1
fi

while ! nc -z "$HOST" "$PORT"; do
  sleep $WAIT
done

$@
