#!/bin/sh

set -eu

REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_CLUSTER_REPLICAS="${REDIS_CLUSTER_REPLICAS:-1}"

_nodes="$(echo "${REDIS_CLUSTER_HOSTS}" | sed -e 's/,/ /g')"

TIMEOUT=120
for name in ${_nodes}; do
	TIMEOUT_END=$(($(date +%s) + TIMEOUT))
	if [ -n "${REDIS_DNS_FORMAT:-}" ]; then
		name="$(printf "${REDIS_DNS_FORMAT:-}" "${name}")"
	fi
	while [ "$(date +%s)" -lt ${TIMEOUT_END} ]; do
		echo "PING ${name}"
		if ! redis-cli --no-auth-warning --pipe-timeout 1 -a "${REDIS_PASSWORD}" -h "${name}" -p "${REDIS_PORT}" ping; then
		 	sleep 1
			continue
		else
		 	echo "${name} is up"
			break
		fi
	done
done

ips=""
for name in ${_nodes}; do
	if [ -n "${REDIS_DNS_FORMAT:-}" ]; then
		name="$(printf "${REDIS_DNS_FORMAT:-}" "${name}")"
	fi

	ip="$(dig +search +short $name)"
	if [ -z "${ip}" ]; then
		echo "Warning could not resolve ${name}"
		continue
	fi
	echo "${name} -> ${ip}"
	ips="$ips $ip:${REDIS_PORT}"
done

echo "creating cluster '${ips}'"
redis-cli --no-auth-warning -a "${REDIS_PASSWORD}" --cluster \
	create $ips \
		--cluster-replicas ${REDIS_CLUSTER_REPLICAS} \
		--cluster-yes
