# Golang builder
FROM golang:1.17.3-alpine as build
RUN mkdir /app
# Download and cache the dependancies
COPY go.mod /app
COPY go.sum /app
WORKDIR /app
RUN go mod download
# Copy source code over
COPY . /app

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen

# Main image
FROM alpine:3.14 as harrybrwn
LABEL maintainer="Harry Brown <harrybrown98@gmail.com>"
# Install timezone data
RUN apk update && \
	apk add tzdata
# copy the binary
COPY --from=build /app/bin/harrybrwn /app/harrybrwn
CMD /app/harrybrwn

FROM alpine:3.14 as user-gen
COPY --from=build /app/bin/user-gen /app/user-gen
CMD /app/user-gen