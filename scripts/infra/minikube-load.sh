#!/bin/sh

set -eu

load() {
  echo "Loading \"${1}\" into minikube..."
  minikube image load "${1}"
  echo "done loading \"${1}\"."
}

images="$(docker-compose --file docker-compose.yml --file config/docker-compose.logging.yml config \
  | grep -E 'image:.*/harrybrwn.*' \
  | awk '{ print $2 }')"
for image in ${images}; do
  image="$(echo "${image}" | sed -Ee 's/(^.*?):(.*$)/\1/')"
  load "${image}" &
done

wait