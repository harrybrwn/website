#!/usr/bin/bash

set -euo pipefail

source "scripts/shell/common.sh"
readonly K3D_CONFIG=config/k8s/k3d.yml
K3D_CLUSTER="$(k3d_cluster_name "${K3D_CONFIG}")"

readarray -t images < <(list-images "$@")
existing_images=()
for i in "${images[@]}"; do
  if docker image inspect "$i" > /dev/null 2>&1; then
    existing_images+=("$i")
  else
    echo "Warning: skipping image \"$i\""
  fi
done

for i in "${images[@]}"; do info util "found image '$i'"; done
info util "Loading images"
k3d image load --cluster "${K3D_CLUSTER}" "${existing_images[@]}"
