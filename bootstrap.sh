#!/usr/bin/bash

set -meuo pipefail

USE_COMPOSE=true
USE_K8s=false
K3D_CLUSTER=hrry-dev

k3d_running() {
  if [ "$(k3d cluster get hrry-dev -o json | jq -M '.[0].serversRunning')" = "1" ]; then
    return 0
  else
    return 1
  fi
}

network_exists() {
  docker network inspect "$1" > /dev/null 2>&1
}

wait_on_dbs() {
  # wait for postgres and minio
  echo "Waiting for postgres and minio to start."
  scripts/wait.sh --timeout 60 --wait 1 \
    tcp://localhost:5432 \
    http://localhost:9000/minio/health/cluster
  sleep 1
}

provision() {
  echo "Provisioning databases."
  bin/provision --config config/provision.json --config config/provision.dev.json
  bin/provision --config config/provision.json --config config/provision.dev.json migrate up --all
}

services=()
while [ $# -gt 0 ]; do
  case $1 in
    --compose)
      USE_COMPOSE=true
      shift 1
      ;;
    --k8s|--k3d|--kube|--kubernetes)
      USE_COMPOSE=false
      USE_K8s=true
      shift 1
      ;;
    --k3d-cluster)
      K3D_CLUSTER="$2"
      shift 2
      ;;
    -h|-help|--help)
      svcs=$(docker compose config --services)
      echo "Usage $0 [flags...] [services...]"
      echo
      echo "Flags"
      echo "  -h --help     print help message"
      echo "     --compose  use 'docker compose'"
      echo "     --k8s      use kubenetes (uses k3d)"
      echo
      echo "Services"
      for s in ${svcs}; do
        echo "  $s"
      done
      exit 0
      ;;
    *)
      services+=("$1")
      shift
      ;;
  esac
done

go build -trimpath -ldflags "-s -w" -o bin/user-gen ./cmd/tools/user-gen
go build -trimpath -ldflags "-s -w" -o bin/provision ./cmd/provision

if ! network_exists "hrry.me"; then
  docker network create hrry.me \
  	--driver "bridge"           \
    --gateway "172.22.0.1"      \
    --subnet "172.22.0.0/16"
fi

if ${USE_COMPOSE}; then
  echo "Starting databases."
  docker compose up --no-deps --detach --force-recreate db s3
elif ${USE_K8s}; then
  if ! k3d cluster get "${K3D_CLUSTER}" > /dev/null; then
    k3d cluster create --config config/k8s/k3d.yml
  elif ! k3d_running; then
    k3d cluster start "${K3D_CLUSTER}"
  fi
  scripts/infra/k3d-load.sh
  k3d kubeconfig merge "${K3D_CLUSTER}" --kubeconfig-merge-default
  kubectl apply -k config/k8s/dev || true # fails sometimes when cert-manager CRDs are being installed.
  kubectl wait pods -l app=db --for condition=Ready
  kubectl wait pods -l app=s3 --for condition=Ready
  kubectl port-forward svc/s3 9000:9000 &
  kubectl port-forward svc/db 5432:5432 &
fi

if [ ! -f config/pki/certs/ca.crt ]; then
  scripts/certs.sh
fi

wait_on_dbs
provision

if ${USE_COMPOSE} && [ ${#services[@]} -gt 0 ]; then
	echo "Starting services \"${services[@]}\"."
  docker compose up --force-recreate --detach "${services[@]}"
elif ${USE_K8s}; then
  kubectl apply -k config/k8s/dev
fi

echo 'testbed' | bin/user-gen - \
  --yes                      \
  --env config/env/db.env    \
  --email 'admin@hrry.local' \
  --username 'admin'         \
  --roles 'admin,family,tanya'

if [ -n "$(jobs -p)" ]; then
  kill $(jobs -p)
fi