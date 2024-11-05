# syntax=docker/dockerfile:1.5

ARG ALPINE_VERSION=3.14
ARG NGINX_VERSION=1.23.3-alpine
ARG NODE_VERSION=16.13.1-alpine
# ARG GO_VERSION=1.18-alpine
ARG GO_VERSION=1.22-alpine
ARG RUST_VERSION=1.71.1

#
# Frontend Build
#
FROM node:${NODE_VERSION} as frontend
ARG TARGETPLATFORM
RUN --mount=type=cache,id=node-apk-${TARGETPLATFORM},target=/var/cache/apk \
    apk update  && \
    apk upgrade && \
    apk add git && \
    mkdir -p /usr/local/sbin/ && \
    ln -s /usr/local/bin/node /usr/local/sbin/node
RUN --mount=type=cache,id=npm-${TARGETPLATFORM},target=/root/.npm \
    npm update -g npm
RUN git clone --depth 1 --branch v1.1.2 \
    https://github.com/harrybrwn/hextris.git /opt/hextris && \
    rm -rf \
    /opt/hextris/.git       \
    /opt/hextris/CNAME      \
    /opt/hextris/README.md  \
    /opt/hextris/.gitignore \
    /opt/hextris/.github && \
    git clone --depth 1 --branch 2.3.3 \
    https://github.com/Joxit/docker-registry-ui.git /opt/docker-registry-ui
# Cache dependancies
WORKDIR /opt/harrybrwn
COPY ./package.json ./yarn.lock tsconfig.json /opt/harrybrwn/
COPY frontend/legacy/ frontend/legacy/
RUN --mount=type=cache,id=yarn-${TARGETPLATFORM},target=/usr/local/share/.cache/yarn \
    --mount=type=cache,id=npm-${TARGETPLATFORM},target=/root/.npm \
    yarn install
#COPY ./frontend/ /opt/harrybrwn/frontend/
RUN --mount=type=cache,id=yarn,target=/usr/local/share/.cache/yarn \
    --mount=type=cache,id=npm,target=/root/.npm \
    yarn workspaces run build
COPY frontend/1k build/1k.hrry.me
COPY ./cmd/hooks/*.html ./cmd/hooks/

#
# Raw Frontend Output
#
FROM scratch as raw-frontend
COPY --from=frontend /opt/harrybrwn/build /

#
# Wait script
#
FROM scratch as wait
COPY ./scripts/wait.sh /bin/wait.sh

#
# Golang builder
#
FROM golang:${GO_VERSION} as builder
RUN --mount=type=cache,id=golang-apk,target=/var/cache/apk \
    apk update && apk add git
RUN --mount=type=cache,id=gobuild,target=/root/.cache/go-build \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    CGO_ENABLED=0 go install -ldflags "-w -s" github.com/golang/mock/mockgen@v1.6.0 && \
    CGO_ENABLED=0 go install -tags 'postgres' -ldflags "-w -s" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1
COPY go.mod go.sum /opt/harrybrwn/
WORKDIR /opt/harrybrwn/
RUN --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go mod download
# build flags
ENV LINK='-s -w'
ENV GOFLAGS='-trimpath'
ENV CGO_ENABLED=0
COPY pkg pkg/
COPY db/migrations db/migrations
COPY files files/
COPY frontend/legacy/templates frontend/legacy/templates/
COPY --from=frontend /opt/harrybrwn/build/harrybrwn.com build/harrybrwn.com/
COPY --from=frontend /opt/harrybrwn/frontend/legacy/embeds.go ./frontend/legacy/embeds.go
COPY --from=frontend /opt/harrybrwn/frontend/legacy/pages ./frontend/legacy/pages

FROM builder as api-builder
COPY cmd/api cmd/api
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/harrybrwn ./cmd/api

FROM builder as legacy-site-builder
COPY cmd/legacy-site cmd/legacy-site
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/legacy-site ./cmd/legacy-site

FROM builder as backups-builder
COPY cmd/backups cmd/backups
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/backups ./cmd/backups

FROM builder as hooks-builder
COPY cmd/hooks cmd/hooks
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/hooks ./cmd/hooks

FROM builder as geoip-builder
COPY services/go-geoip services/go-geoip
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/geoip ./services/go-geoip

FROM builder as vanity-imports-builder
COPY cmd/vanity-imports cmd/vanity-imports
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/vanity-imports ./cmd/vanity-imports

FROM builder as provision-builder
COPY cmd/provision cmd/provision
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/provision ./cmd/provision

FROM builder as registry-auth-builder
COPY cmd/registry-auth cmd/registry-auth
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/registry-auth ./cmd/registry-auth

FROM builder as geoipupdate-go-builder
COPY cmd/geoipupdate/main.go cmd/geoipupdate/main.go
RUN --mount=type=cache,id=gobuild,target=/root/.cache \
    --mount=type=cache,id=gomod,target=/go/pkg/mod \
    go build -ldflags "${LINK}" -o bin/geoipupdate ./cmd/geoipupdate

#
# Base service
#
FROM alpine:${ALPINE_VERSION} as service
RUN apk update && apk upgrade && apk add -l tzdata curl ca-certificates
COPY --from=wait /bin/wait.sh /usr/local/bin/wait.sh

#
# Main image
#
FROM service as api
LABEL maintainer="Harry Brown <harry@harrybrwn.com>"
COPY scripts/wait.sh /usr/local/bin/wait.sh
COPY --from=api-builder /opt/harrybrwn/bin/harrybrwn /app/harrybrwn
WORKDIR /app
ENTRYPOINT ["/app/harrybrwn"]

#
# Build hook server
#
FROM service as hooks
RUN apk update && apk upgrade && apk add -l tzdata
COPY scripts/wait.sh /usr/local/bin/wait.sh
COPY --from=hooks-builder /opt/harrybrwn/bin/hooks /app/hooks
WORKDIR /app
ENTRYPOINT ["/app/hooks"]

#
# Database Backup service
#
FROM service as backups
RUN apk add postgresql-client
COPY --from=backups-builder /opt/harrybrwn/bin/backups /usr/local/bin/
ENTRYPOINT ["backups"]

#
# GeoIP API
#
FROM service as go-geoip
RUN mkdir -p /opt/geoip
COPY files/mmdb/2023-04-27/GeoLite2* /opt/geoip/
COPY --from=geoip-builder /opt/harrybrwn/bin/geoip /usr/local/bin/
ENTRYPOINT ["geoip"]
CMD ["--file=file:///opt/geoip/GeoLite2-City.mmdb", "--file=file:///opt/geoip/GeoLite2-ASN.mmdb"]

#
# Go package vanity imports
#
FROM service as vanity-imports
COPY --from=vanity-imports-builder /opt/harrybrwn/bin/vanity-imports /usr/local/bin/
ENTRYPOINT ["vanity-imports"]

#
# My old website
#
FROM service as legacy-site
COPY --from=frontend /opt/harrybrwn/frontend/legacy/templates /opt/harrybrwn/templates
COPY --from=legacy-site-builder /opt/harrybrwn/bin/legacy-site /usr/local/bin/
ENTRYPOINT ["legacy-site", "--templates", "/opt/harrybrwn/templates"]

#
# geoipupdate-go
#
FROM service as geoipupdate-go
COPY --from=geoipupdate-go-builder /opt/harrybrwn/bin/geoipupdate /usr/local/bin/
ENTRYPOINT [ "geoipupdate" ]

#
# registry auth service
#
FROM service as registry-auth
COPY --from=registry-auth-builder /opt/harrybrwn/bin/registry-auth /usr/local/bin/
ENTRYPOINT ["registry-auth"]

#
# Rust builder
#
FROM --platform=$BUILDPLATFORM harrybrwn/rust:${RUST_VERSION} as rust-builder
ARG BUILDPLATFORM
ARG TARGETPLATFORM
WORKDIR /opt/hrry.me
RUN mkdir -p /usr/local/multi-cargo/${TARGETPLATFORM} && \
    mv /usr/local/cargo/* /usr/local/multi-cargo/${TARGETPLATFORM}/
ENV CARGO_HOME /usr/local/multi-cargo/${TARGETPLATFORM}
ENV PATH=${CARGO_HOME}/bin:${PATH}
RUN --mount=type=cache,target=/var/cache/apk \
    setup-rust
COPY Cargo.toml Cargo.lock ./
COPY services/geoip services/geoip
COPY services/geoipupdate services/geoipupdate
COPY services/gopkg services/gopkg
COPY services/lnsmol services/lnsmol
COPY lib/rust lib/rust
RUN --mount=type=cache,target=/usr/local/multi-cargo/${TARGETPLATFORM}/registry \
    cargo fetch && \
    mkdir -p .cargo && \
    # Vendor for arm builds so they can find the custom assembly in the ring crate
    cargo vendor 2>/dev/null >> .cargo/config.toml
ENV CARGO_INCREMENTAL=1
ENV CARGO_CACHE_RUSTC_INFO=1
ENV CARGO_TARGET_DIR=/opt/hrry.me/target/${TARGETPLATFORM}
RUN --mount=type=cache,target=/usr/local/multi-cargo/${TARGETPLATFORM}/registry \
    --mount=type=cache,target=/opt/hrry.me/target/${TARGETPLATFORM} \
    export TARGET="$(rust-target)" && \
    cargo build --release --target "${TARGET}"
RUN --mount=type=cache,target=/usr/local/multi-cargo/${TARGETPLATFORM}/registry \
    --mount=type=cache,target=/opt/hrry.me/target/${TARGETPLATFORM} \
    export TARGET="$(rust-target)" && \
    ls -la "target" && \
    ls -la "target/${TARGETPLATFORM}" && \
    # ls -la "target/${TARGETPLATFORM}/release" && \
    # ls -la "target/${TARGETPLATFORM}/${TARGET}" && \
    # ls -la "target/${TARGETPLATFORM}/${TARGET}/release/" && \
    for exe in \
        geoip \
        geoipupdate \
        lnsmol \
        gopkg; do \
        mv "target/${TARGETPLATFORM}/${TARGET}/release/${exe}" /usr/local/bin/; \
    done

#######################
# geoipupdate
#######################
FROM alpine:${ALPINE_VERSION} as geoipupdate
RUN apk -U add ca-certificates openssl && rm -rf /var/cache/apk
COPY --from=rust-builder /usr/local/bin/geoipupdate /usr/bin/
ENTRYPOINT [ "geoipupdate" ]

#######################
# geoip
#######################
FROM alpine:${ALPINE_VERSION} as geoip-rs
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=rust-builder /usr/local/bin/geoip /usr/bin/
ENTRYPOINT [ "geoip" ]

#######################
# lnsmol
#######################
FROM alpine:${ALPINE_VERSION} as lnsmol
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=rust-builder /usr/local/bin/lnsmol /usr/bin/
ENTRYPOINT [ "lnsmol" ]

#######################
# gopkg
#######################
FROM alpine:${ALPINE_VERSION} as gopkg-rs
RUN apk -U add ca-certificates && rm -rf /var/cache/apk
COPY --from=rust-builder /usr/local/bin/gopkg /usr/bin/
ENTRYPOINT [ "gopkg" ]

# #######################
# # bk
# #######################
# FROM alpine:${ALPINE_VERSION} as bk
# RUN apk -U add ca-certificates && rm -rf /var/cache/apk
# COPY --from=rust-builder /usr/local/bin/bk /usr/bin/
# ENTRYPOINT [ "bk" ]

#
# mkdocs
#
FROM squidfunk/mkdocs-material:9.5.18 as mkdocs
WORKDIR /opt/hrry.me/
COPY mkdocs.yml ./
COPY docs/ docs/
RUN mkdocs build

#####################
# Webserver Frontend
#####################
FROM harrybrwn/nginx:${NGINX_VERSION} as nginx
ENV REGISTRY_UI_ROOT=/var/www/registry.hrry.dev
RUN --mount=type=cache,id=nginx-apk,target=/var/cache/apk \
    apk update && \
    apk upgrade && \
    apk add ca-certificates
COPY scripts/wait.sh /usr/local/bin/wait.sh
# For some reason update-ca-certificates does not work on arm/v7 so I'm commenting it out for now.
#COPY config/docker-root-ca.pem /usr/local/share/ca-certificates/registry.crt
#RUN update-ca-certificates
COPY --from=frontend /opt/hextris /var/www/hextris.harrybrwn.com
COPY --from=frontend /opt/docker-registry-ui/dist /var/www/registry.hrry.dev/
COPY --from=frontend /opt/docker-registry-ui/favicon.ico /var/www/registry.hrry.dev/
COPY --from=frontend /opt/harrybrwn/build/harrybrwn.com /var/www/harrybrwn.com
COPY --from=harrybrwn/harrybrwn.github.io / /var/www/hrry.me
COPY --from=frontend /opt/harrybrwn/cmd/hooks/index.html /var/www/hooks.harrybrwn.com/index.html
COPY --from=mkdocs /opt/hrry.me/build/docs.hrry.dev /var/www/docs.hrry.dev
COPY config/nginx/docker-entrypoint.sh /docker-entrypoint.sh
COPY config/nginx/ /etc/nginx/
COPY frontend/1k /var/www/1k.hrry.me
COPY frontend/mike /var/www/mike.hrry.me

#
# Registry UI
#
FROM nginx:${NGINX_VERSION} as registry-ui
ENV REGISTRY_UI_ROOT=${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/dist ${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/favicon.ico ${REGISTRY_UI_ROOT}/
COPY config/nginx/docker-entrypoint.sh /docker-entrypoint.sh

#
# Testing tools
#
FROM python:3.9-slim-buster as python
VOLUME /opt/harrybrwn
ARG POETRY_VERSION=1.4.2
ENV PATH="/root/.poetry/bin:$PATH"
ENV POETRY_VIRTUALENVS_CREATE=false
ENV GET_POETRY_IGNORE_DEPRECATION=1
# Tell python's 'requests' to use the system certificates
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
WORKDIR /opt/harrybrwn/test
RUN apt update && \
    apt upgrade -yq && \
    apt install -yq \
    curl netcat build-essential \
    libffi-dev libpq-dev python3-dev \
    vim less && \
    pip install --user --upgrade pip && \
    pip install poetry==$POETRY_VERSION && \
    echo "alias l='ls -lA --group-directories-first --color=always'" >> /root/.bashrc
COPY test/pyproject.toml test/poetry.lock /opt/harrybrwn/test/
RUN poetry install
# Random tools
COPY scripts/migrate.sh scripts/wait.sh /usr/local/bin/
# Install self signed cert
COPY config/pki/certs/ca.crt /usr/local/share/ca-certificates/HarryBrown.crt
RUN update-ca-certificates
# Grab some go tools from our go builder.
COPY --from=builder /go/bin/migrate /usr/local/bin/
WORKDIR /opt/harrybrwn

#
# Provision Tool
#
FROM alpine:${ALPINE_VERSION} as provision
COPY scripts/wait.sh /usr/local/bin/wait.sh
COPY --from=provision-builder /opt/harrybrwn/bin/provision /usr/local/bin/provision
ENTRYPOINT [ "/usr/local/bin/provision" ]

#
# debugging tool
#
FROM builder as debug
RUN apk add bash curl bind-tools
COPY cmd/tools/debug cmd/tools/debug
RUN go build -o /usr/local/bin/debug ./cmd/tools/debug
ENTRYPOINT ["bash"]

#
# DB Tools
#
FROM 10.0.0.11:5000/harrybrwn/postgres:13.6 as data-tools
ARG BUILDPLATFORM
ARG TARGETPLATFORM
ARG MC_VERSION=RELEASE.2023-05-04T18-10-16Z
RUN \
    case "${TARGETPLATFORM}" in \
    linux/amd64) \
    wget -q https://dl.min.io/client/mc/release/linux-amd64/mc.${MC_VERSION} && \
    wget -q https://dl.min.io/client/mc/release/linux-amd64/mc.${MC_VERSION}.sha256sum \
    ;; \
    linux/arm/v7) \
    wget -q https://dl.min.io/client/mc/release/linux-arm/mc.${MC_VERSION} && \
    wget -q https://dl.min.io/client/mc/release/linux-arm/mc.${MC_VERSION}.sha256sum \
    ;;  \
    esac && \
    # https://github.com/gliderlabs/docker-alpine/issues/174
    sed -i 's/ /  /g;' mc.${MC_VERSION}.sha256sum && \
    sha256sum -wc "mc.${MC_VERSION}.sha256sum"    && \
    chmod +x "mc.${MC_VERSION}"                   && \
    mv "mc.${MC_VERSION}" /usr/local/bin/mc       && \
    cat <<EOF > /usr/local/bin/entrypoint.sh
set -eu
if [ -z "\$@" ]; then
    sh
else
    exec "\$@"
fi
EOF
RUN chmod +x /usr/local/bin/entrypoint.sh
ENTRYPOINT [ "sh", "/usr/local/bin/entrypoint.sh" ]
