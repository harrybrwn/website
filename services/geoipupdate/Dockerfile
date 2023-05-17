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
# COPY Cargo.toml Cargo.lock ./
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
	case "${TARGETPLATFORM}" in \
		linux/amd64)  export TARGET='x86_64-unknown-linux-musl' ;;\
		linux/arm64)  export TARGET='aarch64-unknown-linux-musl' ;;\
		linux/arm/v7) export TARGET='armv7-unknown-linux-musleabi' ;;\
		linux/arm/v6) export TARGET='arm-unknown-linux-musleabi' ;;\
		*) \
			echo "Error: ${TARGETPLATFORM} not supported" ;\
			exit 1 ;\
			;; \
	esac && \
	mv "target/${TARGET}/release/geoipupdate" /usr/local/bin/

#######################
# geoipupdate
#######################
FROM alpine:${ALPINE_VERSION}
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=geoipupdate-builder /usr/local/bin/geoipupdate /usr/bin/
CMD [ "geoipupdate" ]
