# syntax=docker/dockerfile:1.5

ARG ALPINE_VERSION=3.16
ARG RUST_VERSION=latest

#
# geoipupdate builder
#
FROM --platform=$BUILDPLATFORM harrybrwn/rust:${RUST_VERSION} as geoipupdate-builder
ARG BUILDPLATFORM
ARG TARGETPLATFORM
WORKDIR /opt/hrry.me/geoipupdate
ENV USER root
RUN setup-rust && \
	cargo init
COPY Cargo.toml ./
RUN --mount=type=cache,target=/usr/local/cargo/registry,id=cargo-registry \
	cargo fetch && \
	mkdir -p .cargo && \
	# Vendor for arm builds so they can find the custom assembly in the ring crate
    cargo vendor 2>/dev/null >> .cargo/config.toml
# COPY services/geoipupdate/src services/geoipupdate/src
COPY src src
RUN --mount=type=cache,target=/usr/local/cargo/registry,id=cargo-registry \
	--mount=type=cache,target=/opt/hrry.me/geoipupdate/target,id=rust-target-${TARGETPLATFORM} \
	cargo build --package geoipupdate --release && \
	export TARGET="$(rust-target)" && \
	mv "target/${TARGET}/release/geoipupdate" /usr/local/bin/

#######################
# geoipupdate
#######################
FROM alpine:${ALPINE_VERSION}
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=geoipupdate-builder /usr/local/bin/geoipupdate /usr/bin/
CMD [ "geoipupdate" ]
