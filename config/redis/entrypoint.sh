#!/bin/sh

set -e

set -- redis-server
if [ -n "${REDIS_CONFIG}" ]; then
	set -- "$@" "${REDIS_CONFIG}"
fi

set -- "$@" --port "${REDIS_PORT:-6379}"

if [ -n "${REDIS_PASSWORD}" ]; then
	set -- "$@" --requirepass "${REDIS_PASSWORD}"
fi

# allow the container to be started with `--user`
if [ "$1" = 'redis-server' -a "$(id -u)" = '0' ]; then
	find . \! -user redis -exec chown redis '{}' +
	exec su-exec redis "$0" "$@"
fi

exec "$@"
