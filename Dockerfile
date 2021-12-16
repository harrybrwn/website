FROM node:16.13.1-alpine as frontend
WORKDIR /app/frontend
RUN stat /app/frontend
RUN apk update && apk upgrade && \
    npm config set update-notifier false && \
    npm update -g npm
#RUN npm -g update -g npm
# Cache dependancies
COPY ./package.json .
COPY ./yarn.lock .
RUN yarn install

# Run the entire build
COPY . .
RUN yarn build

# Golang builder
FROM golang:1.17.3-alpine as build
# Download and cache the dependancies
WORKDIR /app
COPY go.mod /app
COPY go.sum /app
RUN go mod download
# Copy source code over
COPY . .
COPY --from=frontend /app/frontend/build ./build

RUN CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/harrybrwn && \
	CGO_ENABLED=0 go build -trimpath -ldflags "-w -s" -o bin/user-gen ./cmd/user-gen

# Main image
FROM alpine:3.14 as harrybrwn
LABEL maintainer="Harry Brown <harrybrown98@gmail.com>"
RUN apk update && apk upgrade && apk add -l tzdata
COPY --from=build /app/bin/harrybrwn /app/harrybrwn
CMD ["/app/harrybrwn"]

# User Creation Tool
FROM alpine:3.14 as user-gen
COPY --from=build /app/bin/user-gen /app/user-gen
CMD ["/app/user-gen"]
