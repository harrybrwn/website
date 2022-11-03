#!/bin/sh

set -eu

USE_COMPOSE=true
USE_K8s=false

k3d_running() {
  if [ "$(k3d cluster get hrry-dev -o json | jq -M '.[0].serversRunning')" = "1" ]; then
    return 0
  else
    return 1
  fi
}

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
  esac
done

go build -o bin/provision ./cmd/provision

if ${USE_COMPOSE}; then
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

scripts/wait.sh --timeout 60 --wait 1 \
  tcp://localhost:5432 \
  http://localhost:9000/minio/health/cluster

bin/provision --config config/provision.json --config config/provision.dev.json
bin/provision --config config/provision.json --config config/provision.dev.json migrate up --all
