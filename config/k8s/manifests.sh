#!/bin/sh

set -eu

find ./app \
  -type f  \
  \( \
    -name '*.yml' \
    -o -name '*.yaml' \
  \) \
  -not -name 'kustomization.yml' \
  -not -name './app/registry/config.yml'
