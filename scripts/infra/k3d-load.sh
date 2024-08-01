#!/usr/bin/bash

set -euo pipefail

source "scripts/shell/common.sh"
readonly SCRIPT="$0"
readonly K3D_CONFIG=config/k8s/k3d.yml
K3D_CLUSTER="$(k3d_cluster_name "${K3D_CONFIG}")"

usage() {
  echo "Load images in k3d."
  echo
  echo "Usage"
  echo "  $SCRIPT [options]"
  echo
  echo "Options"
  echo "  -h --help   get help message"
  echo "  -l --list   list images"
}

LIST=false
while [ $# -gt 0 ]; do
  case $1 in
    -h|--help)
      usage
      exit 0
      ;;
    -l|--list)
      LIST=true
      shift
      ;;
    *)
      shift
      ;;
  esac
done

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

if ${LIST}; then
  for i in "${images[@]}"; do info k3d-load "$i"; done
else
  for i in "${images[@]}"; do info k3d-load "found image '$i'"; done
  info k3d-load "Loading images \"${existing_images[@]}\""
  if [ ${#existing_images[@]} = 0 ]; then
    error k3d-load "no images"
    exit 1
  fi
  k3d image load --cluster "${K3D_CLUSTER}" "${existing_images[@]}"
fi
