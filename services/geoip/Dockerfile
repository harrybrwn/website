# syntax=docker/dockerfile:1.5

ARG ALPINE_VERSION=3.16
ARG RUST_VERSION=latest

FROM --platform=$BUILDPLATFORM harrybrwn/rust:${RUST_VERSION} as builder
ARG BUILDPLATFORM
ARG TARGETPLATFORM

WORKDIR /opt/hrry.me/geoip
RUN setup-rust
RUN USER=root cargo init
COPY Cargo.toml Cargo.lock ./
RUN --mount=type=cache,target=/usr/local/cargo/registry,id=geoip-cargo-registry \
	cargo fetch && \
	mkdir -p .cargo && \
	# Vendor for arm builds so they can find the custom assembly in the ring crate
    cargo vendor 2>/dev/null >> .cargo/config.toml
COPY src src
RUN --mount=type=cache,target=/usr/local/cargo/registry,id=geoip-cargo-registry \
	--mount=type=cache,target=/opt/hrry.me/geoip/target,id=geoip-build-target-${TARGETPLATFORM} \
	cargo build --release && \
	export TARGET="$(rust-target)" && \
	mv "target/${TARGET}/release/geoip" /usr/local/bin/

FROM alpine:${ALPINE_VERSION}
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=builder /usr/local/bin/geoip /usr/local/bin/
CMD [ "geoip" ]
