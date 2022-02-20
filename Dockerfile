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
# Run the entire build
COPY . .
RUN yarn build

# Golang builder
ARG GO_VERSION=1.17.3-alpine
FROM golang:1.17.3-alpine as builder
WORKDIR /app
COPY go.mod go.sum /app/
RUN go mod download
COPY --from=frontend /app .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen

# Main image
FROM alpine:3.14 as api
LABEL maintainer="Harry Brown <harry@harrybrwn.com>"
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=builder /app/bin/harrybrwn /app/harrybrwn
CMD ["/app/harrybrwn"]

# User Creation Tool
FROM alpine:3.14 as user-gen
COPY --from=builder /app/bin/user-gen /app/user-gen
CMD ["/app/user-gen"]

FROM python:3.9-slim-buster as python
COPY --from=builder /app/bin/user-gen /usr/local/bin/user-gen
COPY --from=frontend /app/test /app/test
WORKDIR /app/test

ENV PATH="/root/.poetry/bin:$PATH"
ENV POETRY_VIRTUALENVS_CREATE=false

RUN apt update && \
    apt upgrade -y && \
    apt install -y curl && \
    pip install --user --upgrade pip
RUN curl -sSL https://raw.githubusercontent.com/python-poetry/poetry/master/get-poetry.py | python -
RUN poetry install

