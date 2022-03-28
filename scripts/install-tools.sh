#!/bin/bash

set -e

TOOLS="${@:-mockgen migrate}"

install() {
  go install -trimpath -ldflags '-w -s' $@
}

for tool in $TOOLS; do
  case $tool in
    mockgen)
      install github.com/golang/mock/mockgen@latest
	  ;;
	migrate|golang-migrate)
	  install -tags 'postgres,github,aws_s3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	  ;;
	stringer)
	  install golang.org/x/tools/cmd/stringer@latest
	  ;;
	golangci-lint)
	  install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	  ;;
	pack)
	  install github.com/buildpacks/pack/cmd/pack@latest
	  ;;
	*)
	  echo "Error: unknown tool $tool"
	  exit 1
  esac
  echo "$tool installed."
done
