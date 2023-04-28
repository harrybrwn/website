#!/usr/bin/bash

set -euo pipefail

DATE="$(date '+%Y-%m-%d')"
DEST=files/mmdb

docker_download() {
	local user="$(id -u):$(id -g)"
	local dir="$(realpath ${DEST}/${DATE})"
	mkdir -p "${dir}"
	docker container run         \
		--rm                       \
		--env-file "config/env/prod/maxmind.env" \
		--env GEOIPUPDATE_FREQUENCY=0 \
		--env GEOIPUPDATE_VERBOSE=1   \
		--env "GEOIPUPDATE_EDITION_IDS=GeoLite2-ASN GeoLite2-City GeoLite2-Country" \
		--volume "${dir}:/usr/share/GeoIP" \
		--entrypoint "sh" \
		ghcr.io/maxmind/geoipupdate:v5.0.4 \
		-c "/usr/bin/entry.sh && chown -R ${user} /usr/share/GeoIP/"
}

native_download() {
	geoipupdate -d "${DEST}/${DATE}"
}

download() {
	docker_download "$@"
}

if [ -d "${DEST}/${DATE}" ]; then
	rm -rf "${DEST}/${DATE}"
fi
if [ -L "${DEST}/latest" ]; then
	rm "${DEST}/latest"
fi

download
ln -s "${DATE}" "${DEST}/latest"
ln -s "../../../${DEST}/latest" "services/geoip/testdata/latest"
ln -s "../../../${DEST}/latest" "services/go-geoip/testdata/latest"

mc cp ${DEST}/${DATE}/*.mmdb "hrry.dev/geoip/${DATE}/"
mc tree --files "hrry.dev/geoip/${DATE}/"

for f in ${DEST}/${DATE}/*.mmdb; do
	echo "s3://s3:9000/geoip/${DATE}/$(basename $f)"
done
