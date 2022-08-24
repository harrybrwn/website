#!/bin/sh

set -eu

load() {
  echo "Loading \"${1}\" into minikube..."
  minikube image load "${1}"
  echo "done loading \"${1}\"."
}

ASYNC=false

images="$(docker-compose --file docker-compose.yml --file config/docker-compose.logging.yml --file config/docker-compose.tools.yml config \
  | grep -E 'image:.*' \
  | awk '{ print $2 }' \
  | sort \
  | uniq)"
for image in ${images}; do
  image="$(echo "${image}" | sed -Ee 's/(^.*?):(.*$)/\1/')"
  if ${ASYNC}; then
    load "${image}" &
  else
    load "${image}"
  fi
done

if ${ASYNC}; then
  wait
fi
