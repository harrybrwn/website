#!/usr/bin/bash

set -euo pipefail

source "scripts/shell/common.sh"
readonly K3D_CONFIG=config/k8s/k3d.yml
K3D_CLUSTER="$(k3d_cluster_name "${K3D_CONFIG}")"

info k3d-load "using cluster \"${K3D_CLUSTER}\""

readarray -t images < <(list-images "$@")
existing_images=()
for i in "${images[@]}"; do
  if docker image inspect "$i" > /dev/null 2>&1; then
    existing_images+=("$i")
  else
    echo "Warning: skipping image \"$i\""
  fi
done

for i in "${images[@]}"; do info k3d-load "found image '$i'"; done
info k3d-load "Loading images \"${existing_images[@]}\""
if [ ${#existing_images[@]} = 0 ]; then
  error k3d-load "no images"
  exit 1
fi
k3d image load --cluster "${K3D_CLUSTER}" "${existing_images[@]}"
