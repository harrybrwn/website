#!/bin/sh

set -eu

#images="$(docker-compose \
#  --file docker-compose.yml \
#  --file config/docker-compose.logging.yml \
#  --file config/docker-compose.tools.yml \
#  config \
#  | grep -E 'image:.*' \
#  | grep 'harrybrwn'   \
#  | awk '{ print $2 }' \
#  | sort \
#  | uniq)"

images="$(docker buildx bake \
  --file config/docker/docker-bake.hcl \
  --print 2>&1 \
  | jq -r '.target[] | .tags[]' \
  | sort \
  | uniq \
  | grep -E 'latest$')"

if [ -n "${*:-}" ]; then
  images="$(echo "${images}" | grep "$@")"
fi

echo "loading images..."
echo "${images}"
k3d image load --cluster hrry-dev "${images}"

