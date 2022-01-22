#!/bin/bash

set -e

if [ ! -f config/heroku.env ]; then
    echo 'no config/heroku.env file'
    exit 1
fi

if [ -z "$1" ]; then
    echo 'no api path'
    exit 1
fi

if [[ $1 != /* ]]; then
    echo "must start with '/'"
    exit 1
fi

source config/heroku.env

curl -X GET https://api.heroku.com$1 \
    -H "Authorization: Bearer $HEROKU_API_TOKEN" \
    -H "Accept: application/vnd.heroku+json; version=3"

# See https://devcenter.heroku.com/articles/platform-api-reference