# syntax=docker/dockerfile:1.3

# Frontend Build
ARG NODE_VERSION=16.13.1-alpine
FROM node:16.13.1-alpine as frontend
RUN apk update  && \
    apk upgrade && \
    apk add git && \
    mkdir -p /usr/local/sbin/ && \
    ln -s /usr/local/bin/node /usr/local/sbin/node && \
    npm update -g npm
RUN git clone --depth 1 --branch v1.1.1 \
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
COPY ./frontend/ /opt/harrybrwn/frontend/
RUN yarn install
COPY public /opt/harrybrwn/public
RUN yarn build
COPY . .

# Golang builder
FROM golang:1.18-alpine as builder
RUN CGO_ENABLED=0 go install -ldflags "-w -s" github.com/golang/mock/mockgen@v1.6.0 && \
    CGO_ENABLED=0 go install -tags 'postgres' -ldflags "-w -s" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1
COPY go.mod go.sum /opt/harrybrwn/
WORKDIR /opt/harrybrwn/
RUN go mod download
COPY pkg pkg/
COPY app app/
COPY cmd cmd/
COPY files files/
COPY internal internal/
COPY main.go .
COPY frontend/templates frontend/templates/
COPY --from=frontend /opt/harrybrwn/build build/
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
    CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/hooks ./cmd/hooks

# Main image
FROM alpine:3.14 as api
LABEL maintainer="Harry Brown <harry@harrybrwn.com>"
RUN apk update && apk upgrade && apk add -l tzdata curl
COPY --from=builder /opt/harrybrwn/bin/harrybrwn /app/harrybrwn
WORKDIR /app
ENTRYPOINT ["/app/harrybrwn"]

# Webserver Frontend
FROM nginx:1.20.2-alpine as nginx
ARG REGISTRY_UI_ROOT
ENV REGISTRY_UI_ROOT=${REGISTRY_UI_ROOT}
RUN apk update && \
    apk upgrade && \
    apk add ca-certificates && \
    rm -rf /var/cache/apk/*
COPY config/docker-root-ca.pem /usr/local/share/ca-certificates/registry.crt
RUN update-ca-certificates
COPY --from=frontend /opt/harrybrwn/build/harrybrwn.com /var/www/harrybrwn.com
COPY --from=frontend /opt/harrybrwn/cmd/hooks/index.html /var/www/hooks.harrybrwn.com/index.html
COPY --from=frontend /opt/hextris /var/www/hextris.harrybrwn.com
COPY --from=frontend /opt/docker-registry-ui/dist ${REGISTRY_UI_ROOT}
COPY --from=frontend /opt/docker-registry-ui/favicon.ico ${REGISTRY_UI_ROOT}/
COPY config/nginx/ /etc/nginx/
COPY config/nginx/docker-entrypoint.sh /docker-entrypoint.sh

# Build hook server
FROM alpine:3.14 as hooks
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=builder /opt/harrybrwn/bin/hooks /app/hooks
WORKDIR /app
ENTRYPOINT ["/app/hooks"]

# Testing tools
FROM python:3.9-slim-buster as python
VOLUME /opt/harrybrwn
ENV PATH="/root/.poetry/bin:$PATH"
ENV POETRY_VIRTUALENVS_CREATE=false
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
