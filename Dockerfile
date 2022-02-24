# Hugo
ARG HUGO_VERSION=0.90.1-ext
FROM klakegg/hugo:${HUGO_VERSION} as hugo
ARG ENVIRONMENT=production
COPY . /app
# RUN hugo --environment ${ENVIRONMENT}

# Frontend Build
ARG NODE_VERSION=16.13.1-alpine
FROM node:16.13.1-alpine as frontend
WORKDIR /app
# TODO install autoreconf
RUN apk update && apk upgrade && \
    npm config set update-notifier false && \
    npm update -g npm
# Cache dependancies
COPY ./package.json ./yarn.lock ./
RUN yarn install
COPY . .
RUN yarn build

# Golang builder
ARG GO_VERSION=1.17.3-alpine
FROM golang:1.17.3-alpine as builder
RUN go install github.com/golang/mock/mockgen@v1.6.0 && \
    go install github.com/golang-migrate/migrate/v4/cmd/migrate@v4.15.1
COPY go.mod go.sum /app/
WORKDIR /app
RUN go mod download

COPY --from=frontend /app .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen

# Main image
FROM alpine:3.14 as api
LABEL maintainer="Harry Brown <harry@harrybrwn.com>"
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=builder /app/bin/harrybrwn /app/harrybrwn
ENTRYPOINT ["/app/harrybrwn"]

# User Creation Tool
FROM alpine:3.14 as user-gen
COPY --from=builder /app/bin/user-gen /app/user-gen
CMD ["/app/user-gen"]

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

COPY test/pyproject.toml test/poetry.lock /app/test/
COPY scripts/wait.sh /usr/local/bin/
WORKDIR /app/test
RUN poetry install

COPY --from=builder /app/bin/user-gen /usr/local/bin/user-gen
VOLUME /app
