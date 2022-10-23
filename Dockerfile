# syntax=docker/dockerfile:1.3
ARG ALPINE_VERSION=3.14

#
# Frontend Build
#
ARG NODE_VERSION=16.13.1-alpine
FROM node:${NODE_VERSION} as frontend
RUN apk update  && \
    apk upgrade && \
    apk add git && \
    mkdir -p /usr/local/sbin/ && \
    ln -s /usr/local/bin/node /usr/local/sbin/node && \
    npm update -g npm
RUN git clone --depth 1 --branch v1.1.2 \
    https://github.com/harrybrwn/hextris.git /opt/hextris && \
    rm -rf \
       /opt/hextris/.git       \
       /opt/hextris/CNAME      \
       /opt/hextris/README.md  \
       /opt/hextris/.gitignore \
       /opt/hextris/.github && \
    git clone --depth 1 --branch main \
        https://github.com/Joxit/docker-registry-ui.git /opt/docker-registry-ui
# Cache dependancies
WORKDIR /opt/harrybrwn
COPY ./package.json ./yarn.lock tsconfig.json /opt/harrybrwn/
COPY frontend/package.json frontend/.nvmrc frontend/
COPY frontend/api frontend/api
RUN yarn install
COPY ./frontend/ /opt/harrybrwn/frontend/
RUN yarn workspaces run build
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
FROM golang:1.18-alpine as builder
RUN apk update && apk add git
RUN CGO_ENABLED=0 go install -ldflags "-w -s" github.com/golang/mock/mockgen@v1.6.0 && \
    CGO_ENABLED=0 go install -tags 'postgres' -ldflags "-w -s" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1
COPY go.mod go.sum /opt/harrybrwn/
WORKDIR /opt/harrybrwn/
RUN go mod download
# build flags
ENV LINK='-s -w'
ENV GOFLAGS='-trimpath'
ENV CGO_ENABLED=0
COPY pkg pkg/
COPY app app/
COPY files files/
COPY internal internal/
COPY main.go .
COPY frontend/templates frontend/templates/
COPY --from=frontend /opt/harrybrwn/build/harrybrwn.com build/harrybrwn.com/
COPY --from=frontend /opt/harrybrwn/frontend/embeds.go ./frontend/embeds.go
COPY --from=frontend /opt/harrybrwn/frontend/pages ./frontend/pages
RUN go build -ldflags "${LINK}" -o bin/harrybrwn

FROM builder as legacy-site-builder
COPY cmd/legacy-site cmd/legacy-site
RUN go build -ldflags "${LINK}" -o bin/legacy-site ./cmd/legacy-site

FROM builder as backups-builder
COPY cmd/backups cmd/backups
RUN go build -ldflags "${LINK}" -o bin/backups ./cmd/backups

FROM builder as hooks-builder
COPY cmd/hooks cmd/hooks
RUN go build -ldflags "${LINK}" -o bin/hooks ./cmd/hooks

FROM builder as geoip-builder
COPY cmd/geoip cmd/geoip
RUN go build -ldflags "${LINK}" -o bin/geoip ./cmd/geoip

FROM builder as vanity-imports-builder
COPY cmd/vanity-imports cmd/vanity-imports
RUN go build -ldflags "${LINK}" -o bin/vanity-imports ./cmd/vanity-imports

FROM builder as provision-builder
COPY cmd/provision cmd/provision
RUN go build -ldflags "${LINK}" -o bin/provision ./cmd/provision

FROM builder as registry-auth-builder
COPY cmd/registry-auth cmd/registry-auth
RUN go build -ldflags "${LINK}" -o bin/registry-auth ./cmd/registry-auth

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
COPY --from=builder /opt/harrybrwn/bin/harrybrwn /app/harrybrwn
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
FROM service as geoip
RUN mkdir -p /opt/geoip
COPY files/mmdb/GeoLite2* /opt/geoip/
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
COPY --from=frontend /opt/harrybrwn/frontend/templates /opt/harrybrwn/templates
COPY --from=legacy-site-builder /opt/harrybrwn/bin/legacy-site /usr/local/bin/
ENTRYPOINT ["legacy-site", "--templates", "/opt/harrybrwn/templates"]

#
# registry auth service
#
FROM service as registry-auth
COPY --from=registry-auth-builder /opt/harrybrwn/bin/registry-auth /usr/local/bin/
ENTRYPOINT ["registry-auth"]

#
# Webserver Frontend
#
ARG NGINX_VERSION
FROM nginx:1.20.2-alpine as nginx
ARG REGISTRY_UI_ROOT
ENV REGISTRY_UI_ROOT=${REGISTRY_UI_ROOT}
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*
COPY scripts/wait.sh /usr/local/bin/wait.sh
# For some reason update-ca-certificates does not work on arm/v7 so I'm commenting it out for now.
#COPY config/docker-root-ca.pem /usr/local/share/ca-certificates/registry.crt
#RUN update-ca-certificates
COPY --from=frontend /opt/hextris /var/www/hextris.harrybrwn.com
COPY --from=frontend /opt/docker-registry-ui/dist ${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/favicon.ico ${REGISTRY_UI_ROOT}/
COPY --from=frontend /opt/harrybrwn/build/harrybrwn.com /var/www/harrybrwn.com
COPY --from=frontend /opt/harrybrwn/cmd/hooks/index.html /var/www/hooks.harrybrwn.com/index.html
COPY config/nginx/docker-entrypoint.sh /docker-entrypoint.sh
COPY config/nginx/ /etc/nginx/

#
# Registry UI
#
ARG NGINX_VERSION
FROM nginx:1.20.2-alpine as registry-ui
ARG REGISTRY_UI_ROOT
ENV REGISTRY_UI_ROOT=${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/dist ${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/favicon.ico ${REGISTRY_UI_ROOT}/
COPY config/nginx/docker-entrypoint.sh /docker-entrypoint.sh

#
# Testing tools
#
FROM python:3.9-slim-buster as python
VOLUME /opt/harrybrwn
ENV PATH="/root/.poetry/bin:$PATH"
ENV POETRY_VIRTUALENVS_CREATE=false
ENV GET_POETRY_IGNORE_DEPRECATION=1
# Tell python's 'requests' to use the system certificates
ENV REQUESTS_CA_BUNDLE=/etc/ssl/certs/ca-certificates.crt
WORKDIR /opt/harrybrwn/test
RUN apt update && \
    apt upgrade -y && \
    apt install -y \
        curl netcat build-essential \
        libffi-dev libpq-dev python3-dev \
        vim less && \
    pip install --user --upgrade pip && \
    curl \
        -sSL \
        https://raw.githubusercontent.com/python-poetry/poetry/master/get-poetry.py | python - && \
    echo "alias l='ls -lA --group-directories-first --color=always'" >> /root/.bashrc
COPY test/pyproject.toml test/poetry.lock /opt/harrybrwn/test/
RUN poetry install
# Random tools
COPY scripts/setup.sh scripts/migrate.sh scripts/wait.sh /usr/local/bin/
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
