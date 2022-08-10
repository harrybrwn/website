#!/bin/sh

set -eu

eval $(minikube docker-env)
docker-compose build
