#!/bin/bash

set -eu

load() {
  echo "Loading \"${1}\" into minikube..."
  minikube image load "${1}"
  echo "done loading \"${1}\"."
}

ASYNC=true
N=4

images="$(docker-compose --file docker-compose.yml --file config/docker-compose.logging.yml --file config/docker-compose.tools.yml config \
  | grep -E 'image:.*' \
  | awk '{ print $2 }' \
  | sort \
  | uniq)"

i=0
for image in ${images}; do
  image="$(echo "${image}" | sed -Ee 's/(^.*?):(.*$)/\1/')"
  if ${ASYNC}; then
    load "${image}" &
  else
    load "${image}"
  fi

  if ${ASYNC} && [ $((i%N)) -eq $((N-1)) ]; then
    wait
  fi
  ((i=i+1))
done

wait
