#!/bin/sh

set -e

mc mirror     \
  --exclude "node_modules/*"  \
  --exclude "bin/*"           \
  --exclude "tests/*"         \
  --exclude ".cache/*"        \
  --exclude "tests/*"         \
  --exclude ".pytest_cache/*" \
  --overwrite  \
  --preserve --remove    \
  ./ hrry/source/github.com/harrybrwn/harrybrwn.com/
