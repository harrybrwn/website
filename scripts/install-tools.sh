#!/bin/sh

set -e

go install github.com/golang/mock/mockgen@latest
go install github.com/golang-migrate/migrate/v4/cmd/migrate@latest
