#!/bin/sh

set -e

#GIT_TAG="$(git describe --tags --abbrev=0)"
#GIT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
GIT_COMMIT="$(git rev-parse HEAD)"
DATE="$(date '+%d-%m-%Y_%H:%M:%S')"

s3_alias=hrry.dev

if [ "$1" = "--tar" ]; then
  tarball="backup-$(date '+%Y-%m-%d').tar.gz"
  tar \
    --exclude="node_modules" \
    --exclude=".git" \
    --exclude="bin" \
    --exclude=".cache" \
    --exclude=".pytest_cache" \
    --exclude="terraform/.terraform" \
    --exclude="files/mmdb" \
    --exclude="*/.terraform" \
    --exclude="__pycache__" \
    --exclude="*.tar.gz" \
    --exclude="*.tgz" \
    --exclude="target" \
    --exclude=".next" \
    -czvf \
    "$tarball" .
    mc cp "$tarball" "r2/storage/hrry.me/$tarball"
else
  mc mirror \
    --exclude "node_modules/*"         \
    --exclude "bin/*"                  \
    --exclude "tests/*"                \
    --exclude ".cache/*"               \
    --exclude ".pytest_cache/*"        \
    --exclude ".tmp/*"                 \
    --exclude "terraform/.terraform/*" \
    --exclude "*.terraform/*" \
    --exclude "files/mmdb/*"           \
    --exclude="target/*" \
    --overwrite                        \
    --preserve --remove    \
    "$@" \
    ./ "${s3_alias}/source/github.com/harrybrwn/harrybrwn.com/${DATE}/${GIT_COMMIT}/"
fi
