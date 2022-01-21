#!/bin/sh

set -e

yarn build
go generate ./...
go build \
    -trimpath \
    -ldflags "-w -s" \
    -o bin/harrybrown.com
