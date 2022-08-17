#!/bin/sh

set -e

#GIT_TAG="$(git describe --tags --abbrev=0)"
GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
GIT_COMMIT="$(git rev-parse HEAD)"
DATE="$(date '+%d-%m-%Y_%H:%M:%S')"

s3_alias=hrry.dev

mc mirror     \
  --exclude "node_modules/*"  \
  --exclude "bin/*"           \
  --exclude "tests/*"         \
  --exclude ".cache/*"        \
  --exclude "tests/*"         \
  --exclude ".pytest_cache/*" \
  --overwrite  \
  --preserve --remove    \
  ./ "${s3_alias}/source/github.com/harrybrwn/harrybrwn.com/${DATE}/${GIT_COMMIT}/"
