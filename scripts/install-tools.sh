#!/bin/bash

set -e

TOOLS="${*:-mockgen migrate}"

i() {
  go install -trimpath -ldflags '-w -s' "$@"
}

for tool in $TOOLS; do
  case $tool in
  mockgen)
    i github.com/golang/mock/mockgen@latest
	  ;;
	mc|minio-client)
		i github.com/minio/mc@latest
		;;
	migrate|golang-migrate)
	  i -tags 'postgres,github,aws_s3' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	  ;;
	stringer)
	  i golang.org/x/tools/cmd/stringer@latest
	  ;;
	golangci-lint)
	  i  github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	  ;;
	pack)
	  i  github.com/buildpacks/pack/cmd/pack@latest
	  ;;
	flarectl|cloudflarectl)
	  i  github.com/cloudflare/cloudflare-go/cmd/flarectl@latest
	  ;;
	*)
	  echo "Error: unknown tool $tool"
	  exit 1
  esac
  echo "$tool installed."
done
