#!/bin/bash

set -eu

images() {
  docker compose                                  \
    --file docker-compose.yml                     \
    --file config/docker-compose.logging.yml      \
    --file config/docker-compose.tools.yml config \
    | grep -E 'image:.*' \
    | grep 'harrybrwn'   \
    | awk '{ print $2 }' \
    | sort \
    | uniq
}

tag="${1}"
if [ -z "${tag}" ]; then
  echo "Error: no tag name given"
  exit 1
fi

if [[ "${tag}" =~ ^v([0-9]\.?){2,3} ]]; then
  :
else
  echo "Error: invalid tag ${tag}"
  exit 1
fi

for image in $(images); do
  bare="$(echo "${image}" | sed -Ee 's/(^.*?):(.*$)/\1/')"
  echo "${bare}:${tag}"
  docker image tag "${image}" "${bare}:${tag}"
done
