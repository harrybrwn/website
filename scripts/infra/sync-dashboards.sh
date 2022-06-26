#!/usr/bin/env bash

set -e

env_file=config/env/prod/grafana-admin.env
if [ ! -f  $env_file ]; then
	echo "Error: could not find grafana admin credentials"
	exit 1
fi
source $env_file

conf="./config/logging/grafana/dashboards"
base="${GRAFANA_URL:-http://grafana:3000}"
auth="Authorization: Bearer ${GRAFANA_API_KEY}"

sync_dashboard() {
	local uid="${1}"
	local name="${2}"
	local data="$(curl -sS --fail -H "${auth}" "${base}/api/dashboards/uid/${uid}")"
	if [ $? -ne 0 ]; then
		echo "Error: failed to fetch dashobard"
		exit 1
	fi
  echo "Syncing ${uid} => ${name}.json"
	echo "${data}" | jq '.dashboard' -cM > "${conf}/${name}.json"
}

NGINX_UID=MsjffzSZz
PERF_UID=Tt6NZyXnz
REGISTRY_UID=X9CPPsj7k
GEOIP_UID=4Ba84XCnz
OBS_UID=n-VhV_j7z
MINIO_UID=TgmJnqnnk

sync_dashboard ${NGINX_UID} "nginx"
sync_dashboard ${PERF_UID} "performance"
sync_dashboard ${REGISTRY_UID} "registry"
sync_dashboard ${GEOIP_UID} "geoip"
sync_dashboard ${OBS_UID} "observability-meta"
#sync_dashboard ${MINIO_UID} "minio"
