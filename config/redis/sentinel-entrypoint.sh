#!/bin/sh

set -eu

# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/lib.sh

find_master_node() {
	_MASTER=""
  _nodes="$(echo "${REDIS_SENTINEL_REDIS_HOSTS}" | sed -e 's/,/ /g')"
	for node in ${_nodes}; do
		host="$(echo "${node}" | cut -d ':' -f1)"
		port="$(echo "${node}" | cut -d ':' -f1)"
		if [ -z "${host}" ]; then
			error "cannot ping empty redis node hostname"
			return 1
		fi
		if [ -z "${port}" ] || [ "${port}" = "${host}" ]; then
			port="${REDIS_PORT}"
		fi
		_MASTER="$(redis-cli --no-auth-warning \
			--raw -h "${host}" -p "${port}" -a "${REDIS_PASSWORD}" \
			info replication \
			| awk '{print $1}' \
			| grep 'master_host:' \
			| cut -d ':' -f2)"
		if [ -z "${_MASTER}" ]; then
			continue
		else
			echo "${_MASTER}:${port}"
			return 0
		fi
	done
	error "failed to find master from cluster ${REDIS_SENTINEL_REDIS_HOSTS}"
	return 1
}

# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/redis-env.sh
# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/sentinel-env.sh

# validate variables
if [ -z "${REDIS_SENTINEL_MASTER_NAME}" ]; then
	fatal "no redis cluster master name given. set REDIS_SENTINEL_MASTER_NAME"
	exit 1
elif [ -z "${REDIS_SENTINEL_REDIS_HOSTS:-}" ]; then
	fatal "no redis cluster hosts given. set REDIS_SENTINEL_REDIS_HOSTS as a comma separated list of hostnames or IPs"
	exit 1
elif [ -z "${REDIS_SENTINEL_QUORUM}" ]; then
	fatal "no redis sentinel quorum given. set REDIS_SENTINEL_QUORUM."
	exit 1
fi

# init configuration
if [ ! -d "${REDIS_SENTINEL_CONFIG_DIR}" ]; then
	mkdir -p "${REDIS_SENTINEL_CONFIG_DIR}"
fi

WAIT=1
TIMEOUT=30
TIMEOUT_END=$(($(date +%s) + TIMEOUT))

MASTER=""
while [ "$(date +%s)" -lt ${TIMEOUT_END} ]; do
	MASTER="$(find_master_node)"
  # shellcheck disable=SC2181
	if [ $? -ne 0 ]; then
		sleep ${WAIT}
		continue
	else
		log "master node found ${MASTER}"
		break
	fi
done
if [ -z "${MASTER}" ]; then
	fatal "failed to find master node"
	exit 1
fi
master_host="$(echo "${MASTER}" | cut -d ':' -f1)"
master_port="$(echo "${MASTER}" | cut -d ':' -f2)"
if [ "${master_port}" = "${master_host}" ]; then
	master_port="${REDIS_PORT}"
fi

cat > "${REDIS_SENTINEL_CONFIG_FILE}" <<EOF
port ${REDIS_SENTINEL_PORT}
sentinel resolve-hostnames ${REDIS_SENTINEL_RESOLVE_HOSTNAMES}
sentinel announce-hostnames ${REDIS_SENTINEL_ANNOUNCE_HOSTNAMES}
sentinel monitor ${REDIS_SENTINEL_MASTER_NAME} ${master_host} ${master_port} ${REDIS_SENTINEL_QUORUM}
sentinel down-after-milliseconds ${REDIS_SENTINEL_MASTER_NAME} ${REDIS_SENTINEL_DOWN_AFTER_MILLISECONDS}
$([ -n "${REDIS_SENTINEL_FAILOVER_TIMEOUT}" ] && echo "sentinel failover-timeout ${REDIS_SENTINEL_MASTER_NAME} ${REDIS_SENTINEL_FAILOVER_TIMEOUT}")
sentinel parallel-syncs ${REDIS_SENTINEL_MASTER_NAME} ${REDIS_SENTINEL_PARALLEL_SYNCS}
sentinel auth-pass ${REDIS_SENTINEL_MASTER_NAME} ${REDIS_PASSWORD}
EOF

exec redis-sentinel "${REDIS_SENTINEL_CONFIG_FILE}" "$@"
