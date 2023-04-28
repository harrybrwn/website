# syntax=docker/dockerfile:1.4

ARG RUST_VERSION=1.68.0
ARG RUST_BASE=alpine3.16
FROM --platform=$BUILDPLATFORM rust:${RUST_VERSION}-${RUST_BASE}

ENV PATH=/usr/local/bin/:$PATH
ARG BUILDPLATFORM
ARG TARGETPLATFORM

COPY rust-target /usr/local/bin/
COPY setup-rust /usr/local/bin/
