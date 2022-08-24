#!/bin/bash

set -eu

if [ -z "${SUDO_USER:-}" -o -z "${SUDO_COMMAND:-}" ]; then
  echo "Error: must run script as sudo"
  exit 1
fi

export KUBECONFIG="/home/${SUDO_USER:-}/.kube/config"

pids=()

kubectl port-forward svc/nginx 443:443 80:80 &
pids+=($!)
kubectl port-forward svc/s3 9000:9000 9001:9001 &
pids+=($!)
kubectl port-forward svc/db 5432:5432 &
pids+=($!)
kubectl port-forward svc/grafana 3000:3000 &
pids+=($!)

echo "running ${pids[@]}"
echo "to stop, press <enter>"
read -r line
for pid in "${pids[@]}"; do
  echo "killing ${pid}"
  kill -9 "${pid}"
done
