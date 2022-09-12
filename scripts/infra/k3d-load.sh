#!/bin/sh

set -eu

images="$(docker-compose \
  --file docker-compose.yml \
  --file config/docker-compose.logging.yml \
  --file config/docker-compose.tools.yml \
  config \
  | grep -E 'image:.*' \
  | grep 'harrybrwn'   \
  | awk '{ print $2 }' \
  | sort \
  | uniq)"

echo "loading images..."
echo "${images}"
k3d image load --cluster hrry ${images}
