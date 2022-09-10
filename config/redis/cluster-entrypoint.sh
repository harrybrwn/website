#!/bin/sh

set -eu

# Basic redis config
# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/redis-env.sh
# shellcheck disable=SC1091
. /usr/local/share/redis/scripts/lib.sh

apply_replication_config() {
	if [ -z "${REDIS_DEFAULT_MASTER_DNS:-}" ]; then
		# WARNING this regex depends on the statefulset and service names in kubernetes!
		MASTER_FDQN="$(hostname  -f | sed -e 's/redis-[0-9]\./redis-0./')"
	else
		MASTER_FDQN="${REDIS_DEFAULT_MASTER_DNS}"
	fi

	if [ "$(redis-cli --pipe-timeout 2 -h ${REDIS_SENTINEL_HOST} ping)" != "PONG" ]; then
		if [ "${REDIS_DEFAULT_MASTER:-}" != "true" ] && [ "${HOSTNAME}" != "${REDIS_DEFAULT_MASTER_DNS:-}" ]; then
			echo "slaveof ${MASTER_FDQN} ${REDIS_PORT}" >> "${REDIS_CONFIG_FILE}"
		else
			:
		fi
	else
		MASTER="$(redis-cli -h sentinel -p ${SENTINEL_PORT} sentinel get-master-addr-by-name ${REDIS_SENTINEL_MASTER_NAME} | grep -E '(^redis-\d{1,})|([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})')"
		echo "slaveof ${MASTER} ${REDIS_PORT}" >> "${REDIS_CONFIG_FILE}"
		unset MASTER
	fi

	# Find the default
	announce_ip="${HOSTNAME}"
	if [ -n "${REDIS_BASE_DOMAIN:-}" ]; then
		announce_ip="${HOSTNAME}.${REDIS_BASE_DOMAIN}"
	fi
	cat >> "${REDIS_CONFIG_FILE}" <<EOF
replica-announce-ip   ${announce_ip}
replica-announce-port ${REDIS_PORT}
EOF
	unset announce_ip MASTER_FDQN
}

apply_cluster_config() {
	sleep "$1"
	if [ -z "${REDIS_CLUSTER_HOSTS}" ]; then
		error "no list of cluster nodes. set REDIS_CLUSTER_HOSTS"
		exit 1
	fi

  _nodes="$(echo "${REDIS_CLUSTER_HOSTS}" | sed -e 's/,/ /g')"
	hosts=""
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
		if [ -n "${REDIS_DNS_FORMAT:-}" ]; then
			host="$(printf "${REDIS_DNS_FORMAT:-}" "${host}")"
		fi
		# if [ -n "${REDIS_BASE_DOMAIN:-}" ]; then
		# 	host="${host}.${REDIS_BASE_DOMAIN}"
		# fi

		ip="$(dig +short "${host}")"
		if [ -z "${ip}" ]; then
			continue
		fi

		if ! redis-cli --no-auth-warning -a "${REDIS_PASSWORD}" CLUSTER MEET "${ip}" "${port}"; then
			continue
		else
			break
		fi
	done
}

# init configuration
if [ ! -d "${REDIS_CONFIG_DIR}" ]; then
	mkdir -p "${REDIS_CONFIG_DIR}"
fi

cat > "${REDIS_CONFIG_FILE}" <<EOF
port            ${REDIS_PORT}
dir             "${REDIS_DIR}"
dbfilename      "${REDIS_DBFILENAME}"
appendonly      ${REDIS_APPENDONLY}
appendfilename  "${REDIS_APPENDFILENAME}"
masterauth      ${REDIS_PASSWORD}
requirepass     ${REDIS_PASSWORD}
cluster-enabled ${REDIS_CLUSTER_ENABLED}
cluster-config-file  cluster.conf
cluster-node-timeout ${REDIS_CLUSTER_NODE_TIMEOUT}
EOF


if [ "${REDIS_CLUSTER_ENABLED}" = "yes" ]; then
	echo "protected-mode no" >> "${REDIS_CONFIG_FILE}"
else
	apply_replication_config
fi

exec redis-server "${REDIS_CONFIG_FILE}" "$@"
