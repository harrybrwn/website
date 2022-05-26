#!/bin/sh

set -e

if [ "${REGISTRY_AUTH}" = "htpasswd" -a ! -f "${REGISTRY_AUTH_HTPASSWD_PATH}" ]; then
	:
fi

case "$1" in
    *.yaml|*.yml) set -- registry serve "$@" ;;
    serve|garbage-collect|help|-*) set -- registry "$@" ;;
esac

exec "$@"