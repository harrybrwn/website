ARG GO_VERSION
ARG ALPINE_VERSION

FROM golang:${GO_VERSION}-alpine as builder
WORKDIR /opt/geoipupdate
ENV CGO_ENABLED=0
COPY go.mod go.sum ./
RUN --mount=type=cache,id=gomod-geoipupdate,target=/go/pkg/mod \
    go mod download
COPY . .
RUN --mount=type=cache,id=gomod-geoipupdate,target=/go/pkg/mod \
	--mount=type=cache,target=/root/.cache/go-build \
	go build -trimpath -ldflags '-s -w' -o bin/geoipupdate

FROM alpine:${ALPINE_VERSION}
COPY --from=builder /opt/geoipupdate/bin/geoipupdate /usr/local/bin/
ENTRYPOINT [ "geoipupdate" ]
