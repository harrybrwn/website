#!/bin/bash

set -e

scripts/wait.sh "$POSTGRES_HOST" "$POSTGRES_PORT" -w 1 -- scripts/migrate.sh -env none -- up
scripts/wait.sh "$API_HOST" "$API_PORT" -w 1 -- pytest
