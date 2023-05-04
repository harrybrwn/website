#!/usr/bin/bash

set -euo pipefail

find_image() {
  local e match="$1"
  shift
  for e; do
    if [[ "$e" =~ $match ]]; then
      echo "$e"
      return 0
    fi
  done
  return 1
}

readarray -t images < <(docker buildx bake \
  --file config/docker/docker-bake.hcl \
  --print 2>&1 \
  | jq -r '.target[] | .tags[]' \
  | sort \
  | uniq \
  | grep -E 'latest$')

if [ -n "${*:-}" ]; then
  img="$(find_image "$@" "${images[@]}")"
  images=("$img")
fi

existing_images=()
for i in "${images[@]}"; do
  if docker image inspect "$i" > /dev/null 2>&1; then
    existing_images+=("$i")
  else
    echo "Warning: skipping image \"$i\""
  fi
done

echo "loading images..."
for i in "${images[@]}"; do echo "$i"; done
k3d image load --cluster hrry-dev "${existing_images[@]}"
