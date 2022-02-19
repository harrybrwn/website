#!/bin/sh

set -e

go install github.com/golang-migrate/migrate/cmd/migrate@latest
go install github.com/golang/mock/mockgen@latest

