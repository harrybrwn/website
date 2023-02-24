#!/bin/bash

set -euo pipefail

USE_COMPOSE=true
USE_K8s=false

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

services=()
while [ $# -gt 0 ]; do
  case $1 in
    --compose)
      USE_COMPOSE=true
      shift 1
      ;;
    --k8s|--kube|--kubernetes)
      USE_COMPOSE=false
      USE_K8s=true
      shift 1
      echo "kubernetes config is not supported at the monment"
      exit 1
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
  if ! k3d cluster get hrry-dev > /dev/null; then
    k3d cluster create --config config/k8s/k3d.yml
  elif ! k3d_running; then
    k3d cluster start hrry-dev
  fi
  scripts/infra/k3d-load.sh
  kubectl apply -k config/k8s/dev
fi

if [ ! -f config/pki/certs/ca.crt ]; then
  scripts/certs.sh
fi

# wait for postgres and minio
echo "Waiting for postgres and minio to start."
scripts/wait.sh --timeout 60 --wait 1 \
  tcp://localhost:5432 \
  http://localhost:9000/minio/health/cluster
sleep 1

echo "Provisioning databases."
bin/provision --config config/provision.json --config config/provision.dev.json
bin/provision --config config/provision.json --config config/provision.dev.json migrate up --all

if [ ${#services[@]} -gt 0 ]; then
	echo "Starting services \"${services[@]}\"."
  docker compose up --force-recreate --detach "${services[@]}"
fi

echo 'testbed' | bin/user-gen - \
  --yes                      \
  --env config/env/db.env    \
  --email 'admin@hrry.local' \
  --username 'admin'         \
  --roles 'admin,family,tanya'
