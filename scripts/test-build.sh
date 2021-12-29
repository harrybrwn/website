#!/bin/sh

set -e

pack build testbuild-harrybrwn.com \
    --buildpack heroku/nodejs@0.3.6 \
    --buildpack heroku/go@0.3.1 \
    --builder heroku/buildpacks:20
