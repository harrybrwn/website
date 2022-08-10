#!/bin/bash

set -eu

sudo kubectl --kubeconfig ~/.kube/config port-forward svc/nginx-service 443:443 80:80
