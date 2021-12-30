#!/bin/bash

set -e

OUTDIR=diagrams

drawio \
	--export \
	--output "$OUTDIR/remora.svg" \
	--transparent \
	-f svg \
	--border 5 \
	diagrams/remora.drawio

