#!/bin/sh

set -eu

# Basic redis config
# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/redis-env.sh

# sentinel info for cluster node
REDIS_SENTINEL_MASTER_NAME="${REDIS_SENTINEL_MASTER_NAME:-}"
REDIS_SENTINEL_HOST="${REDIS_SENTINEL_HOST:-}"
REDIS_SENTINEL_PORT="${REDIS_SENTINEL_PORT:-}"
