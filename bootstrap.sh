#!/usr/bin/bash

set -meuo pipefail

source "scripts/shell/common.sh"

USE_K8s=true
USE_COMPOSE=false
readonly K3D_CONFIG=config/k8s/k3d.yml
readonly MC_ALIAS=hrrylocal

network_exists() {
  docker network inspect "$1" > /dev/null 2>&1
}

volume_exists() {
  docker volume inspect "$1" > /dev/null 2>&1
}

do_kubectl() {
  while read -r line; do
    info kubectl "$line"
  done < <(kubectl "$@" 2>&1)
}

wait_on_dbs() {
  # wait for postgres and minio
  echo "Waiting for postgres and minio to start."
  scripts/wait.sh --timeout 60 --wait 1 \
    tcp://localhost:5432 \
    http://localhost:9000/minio/health/cluster
  sleep 1
}

wait_for_k8s_cert_manager() {
  # Wait for CRDs
  do_kubectl wait crds certificates.cert-manager.io --for=condition=Established
  do_kubectl wait crds certificates.cert-manager.io --for=condition=NamesAccepted
  # Wait for the helm chart to complete
  do_kubectl -n kube-system wait jobs -l 'helmcharts.helm.cattle.io/chart=cert-manager' --for=condition=Complete
  # wait for the pods to start
  for name in "cainjector" "webhook" "cert-manager"; do
    do_kubectl -n cert-manager wait pods \
      --for=condition=Ready -l "app.kubernetes.io/name=$name"
  done
}

wait_for_rook() {
  do_kubectl -n rook-ceph wait pods --for=condition=Ready -l 'app=rook-ceph-operator'
  do_kubectl -n rook-ceph wait pods --for=condition=Ready -l 'app.kubernetes.io/name=ceph-mon'
  do_kubectl -n rook-ceph wait pods --for=condition=Ready -l 'app=csi-rbdplugin-provisioner'
}

wait_for_traefik() {
  do_kubectl -n kube-system wait jobs -l 'helmcharts.helm.cattle.io/chart=traefik' --for=condition=Complete
  do_kubectl -n kube-system wait pods -l 'app.kubernetes.io/name=traefik' --for=condition=Ready
}

wait_for_longhorn() {
  do_kubectl -n longhorn wait pods \
    --for=condition=Ready \
    --selector 'app.kubernetes.io/name=longhorn' \
    --timeout '60m'
  for app in "csi-attacher" "csi-resizer" "csi-snapshotter" "csi-provisioner"; do
    do_kubectl -n longhorn wait pods \
      --for=condition=Ready \
      --selector "app=$app" \
      --timeout '60m'
  done
}

start_k3d() {
  local cluster="$1"
  info bootstrap "starting k3d cluster \"${cluster}\""
  if ! k3d cluster get "${cluster}" > /dev/null 2>&1; then
    # if volume_exists "k3d-${cluster}-images"; then
    #   docker volume rm "k3d-${cluster}-images"
    # fi
    if network_exists "k3d-${cluster}"; then
      docker network rm "k3d-${cluster}"
    fi
    info bootstrap "Creating new cluster"
    k3d cluster create --config config/k8s/k3d.yml
    scripts/infra/k3d-load.sh
  elif ! k3d_running "${cluster}"; then
    info bootstrap "Starting existing cluster \"${cluster}\""
    k3d cluster start "${cluster}"
  fi
}

provision() {
  info bootstrap "Provisioning databases."
  bin/provision --config config/provision.json --config config/provision.dev.json
  bin/provision --config config/provision.json --config config/provision.dev.json migrate up --all
}

setup_mc() {
  if ! mc alias ls "${MC_ALIAS}" > /dev/null 2>&1; then
    echo "Creating alias ${MC_ALIAS}..."
    mc alias set "${MC_ALIAS}" \
      "http://localhost:9000" 'root' 'minio-testbed001' \
      --api 's3v4'
  fi
}

upload_mmdb() {
  info bootstrap "Uploading geoip data"
  if [ -d files/mmdb/latest ]; then
    for f in files/mmdb/latest/*.mmdb; do
      #mc cp "$f" "${MC_ALIAS}/files/maxmind/latest/$(basename "$f")"
      mc cp "$f" "${MC_ALIAS}/geoip/latest/$(basename "$f")"
    done
  fi
}

stop_all() {
  info bootstrap "Cleaning up background jobs"
  if [ -n "$(jobs -p)" ]; then
    kill $(jobs -p)
  fi
}

env_files=(
  .env
  secrets.env
)

for env in ${env_files}; do
  if [ -f "${env}" ]; then
    source "${env}"
  fi
done

K3D_CLUSTER="$(k3d_cluster_name "${K3D_CONFIG}")"
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
      echo "     --compose  use 'docker compose' (default: $USE_COMPOSE)"
      echo "     --k8s      use kubenetes (k3d)  (default: $USE_K8s)"
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

# Create certificates
if ! scripts/certs.sh --check ; then
  scripts/certs.sh
fi

# Check configuration
bash scripts/configure.sh

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
  start_k3d "${K3D_CLUSTER}"
  k3d kubeconfig merge "${K3D_CLUSTER}" --kubeconfig-merge-default
  if [ "$(kubectl config current-context)" != "k3d-${K3D_CLUSTER}" ]; then
    kubectl config use-context "k3d-${K3D_CLUSTER}"
  fi
  do_kubectl -n kube-system wait pods -l 'k8s-app=kube-dns' --for=condition=Ready
  do_kubectl -n kube-system wait pods -l 'k8s-app=metrics-server' --for=condition=Ready
  wait_for_traefik
  kubeoutput="$(kubectl kustomize config/k8s/dev 2>&1)"
  if [ $? -ne 0 ]; then
    error bootstrap "kustomize failed: ${kubeoutput}"
    exit 1
  fi

  info bootstrap "Starting helm charts"
  #do_kubectl patch storageclass local-path -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"false"}}}'
  #do_kubectl apply -k config/k8s/common/crds
  do_kubectl apply -k config/k8s/dev/system
  #do_kubectl apply -k config/k8s/common/helm-charts # get the CRDs first
  #wait_for_k8s_cert_manager # wait for cert manager to start...
  do_kubectl apply -k config/k8s/dev | grep -Ev 'unchanged$'
  do_kubectl wait pods -l 'app=db' --for condition=Ready
  do_kubectl wait pods -l 'app=s3' --for condition=Ready
  # Expose databases to localhost
  kubectl port-forward svc/s3 9000:9000 &
  kubectl port-forward svc/db 5432:5432 &
  trap stop_all EXIT
  # Create an admin user for mastodon
  # TODO make this idempotent
  #kubectl -n mastodon wait pods -l 'app.kubernetes.io/name=mastodon' --for condition=Ready
  #kubectl -n mastodon exec -it deployment/mastodon-web -- \
  # tootctl accounts create \
  #   'admin' \
  #   --email 'admin@hrry.local' \
  #   --role Owner \
  #   --confirmed
fi

wait_on_dbs
provision
setup_mc
upload_mmdb

if ${USE_COMPOSE} && [ ${#services[@]} -gt 0 ]; then
	echo "Starting services \"${services[@]}\"."
  docker compose up --force-recreate --detach "${services[@]}"
elif ${USE_K8s}; then
  kubectl apply -k config/k8s/dev > /dev/null
fi

info bootstrap "Creating default admin user"
echo 'testbed' | bin/user-gen - \
  --yes                      \
  --env config/env/db.env    \
  --email 'admin@hrry.local' \
  --username 'admin'         \
  --roles 'admin,family,tanya'
