# Hugo
ARG HUGO_VERSION=0.90.1-ext
FROM klakegg/hugo:${HUGO_VERSION} as hugo
ARG ENVIRONMENT=production
COPY . /app
# RUN hugo --environment ${ENVIRONMENT}

# Frontend Build
ARG NODE_VERSION=16.13.1-alpine
FROM node:16.13.1-alpine as frontend
RUN apk update && apk upgrade && \
    npm config set update-notifier false && \
    npm update -g npm
# Cache dependancies
WORKDIR /opt/harrybrwn
COPY ./package.json ./yarn.lock /opt/harrybrwn/
COPY ./frontend/ /opt/harrybrwn/frontend/
RUN yarn install
COPY . .
RUN yarn build

# Golang builder
FROM golang:1.18-alpine as builder
RUN CGO_ENABLED=0 go install -ldflags "-w -s" github.com/golang/mock/mockgen@v1.6.0 && \
    CGO_ENABLED=0 go install -tags 'postgres' -ldflags "-w -s" github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1
COPY go.mod go.sum /opt/harrybrwn/
WORKDIR /opt/harrybrwn
RUN go mod download

COPY --from=frontend /opt/harrybrwn .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/pwhash ./cmd/pwhash

# Main image
FROM alpine:3.14 as api
LABEL maintainer="Harry Brown <harry@harrybrwn.com>"
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=builder /opt/harrybrwn/bin/harrybrwn /app/harrybrwn
ENTRYPOINT ["/app/harrybrwn"]

# Testing tools
FROM python:3.9-slim-buster as python
ENV PATH="/root/.poetry/bin:$PATH"
ENV POETRY_VIRTUALENVS_CREATE=false
RUN apt update && \
    apt upgrade -y && \
    apt install -y curl netcat build-essential libffi-dev libpq-dev python3-dev && \
    pip install --user --upgrade pip && \
    curl \
        -sSL \
        https://raw.githubusercontent.com/python-poetry/poetry/master/get-poetry.py | python -

COPY test/pyproject.toml test/poetry.lock /opt/harrybrwn/test/
COPY scripts/migrate.sh scripts/wait.sh /usr/local/bin/
WORKDIR /opt/harrybrwn/test
RUN poetry install
# Grabe some go tools from our go builder.
COPY --from=builder /go/bin/migrate /opt/harrybrwn/bin/user-gen /usr/local/bin/
VOLUME /opt/harrybrwn
WORKDIR /opt/harrybrwn
