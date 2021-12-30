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
COPY --from=frontend /app/build .

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen

# Main image
FROM alpine:3.14 as harrybrwn
LABEL maintainer="Harry Brown <harrybrown98@gmail.com>"
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=builder /app/bin/harrybrwn /app/harrybrwn
CMD ["/app/harrybrwn"]

# User Creation Tool
FROM alpine:3.14 as user-gen
COPY --from=builder /app/bin/user-gen /app/user-gen
CMD ["/app/user-gen"]
