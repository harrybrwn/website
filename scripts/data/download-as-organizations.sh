#!/bin/sh

set -eu

base="https://publicdata.caida.org/datasets/as-organizations"

wget -r -p -k "$base"
