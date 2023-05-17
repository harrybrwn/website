#!/bin/bash

set -eu

usage() {
  echo "Usage"
  echo "  expose-k8s [name...] [flags...]"
  echo
  echo "Names"
  echo "  db, postgres"
  echo "  s3, minio"
  echo "  redis"
  echo
  echo "Flags"
  echo "  -h, --help   print usage"
}

if [ -z "${SUDO_USER:-}" ] || [ -z "${SUDO_COMMAND:-}" ]; then
  echo "Error: must run script as sudo"
  exit 1
fi

export KUBECONFIG="/home/${SUDO_USER:-}/.kube/config"

if [ -z "$*" ]; then
  kubectl port-forward svc/nginx 443:443 80:80 &
  kubectl port-forward svc/s3 9000:9000 &
  kubectl port-forward svc/db 5432:5432 &
  kubectl port-forward svc/grafana 3000:3000 &
  kubectl port-forward svc/redis 6379:6379 &
  kubectl port-forward svc/geoip 8084:8084 &
else
  while [ $# -gt 0 ]; do
    case "$1" in
      -h|--help)
        usage
        exit 0
        ;;
      db|postgres)
        kubectl port-forward svc/db 5432:5432 &
        ;;
      s3|minio)
        kubectl port-forward svc/s3 9000:9000 &
        ;;
      redis)
        kubectl port-forward svc/redis 6379:6379 &
        ;;
    esac
    shift
  done
fi

echo "running $(jobs -p)"

stopall() {
  kill -9 "$(jobs -p)"
}
trap stopall SIGINT
wait

# echo "to stop, press <enter>"
# read -n1 line
# stopall
