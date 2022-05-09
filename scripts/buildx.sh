#!/bin/bash

set -e

BUILDKIT_NAME=harrybrwn-builder
BUILDKIT_CONFIG=./config/buildkit-config.toml
REGISTY=registry.digitalocean.com/webreef
# PLATFORMS=linux/amd64,linux/386,linux/arm/v6,linux/arm/v7,linux/arm64,linux/riscv64
PLATFORMS=linux/amd64,linux/arm/v7,linux/arm/v6
IMAGE=
PUSH=false
CACERT=''

CONTEXT=.
DOCKERFILE=./Dockerfile

while [[ $# -gt 0 ]]; do
    case $1 in
    --help|-h)
        echo "Usage:

    $0 [flags] <build context>

Flags:

    -t, --tag       docker image build tag            (default: '$IMAGE')
    -f, --file      docker file                       (default: '$DOCKERFILE')
        --name      name of buildkit build system     (default: '$BUILDKIT_NAME')
        --config    give buildkit a config file       (default: '$BUILDKIT_CONFIG')
        --registry  push to a spesific registry       (default: '$REGISTRY')
        --platform  comma separated list of platforms (default: '$PLATFORMS')
        --push      push to the registry after build  (default: '$PUSH')
        --cacert    ca certificate for pushing to registries
    -h, --help      print help message"
        exit 0
        ;;
    --name)
        BUILDKIT_NAME="$2"
        shift 2
        ;;
    --platform)
        PLATFORMS="$2"
        shift 2
        ;;
    --config)
        BUILDKIT_CONFIG="$2"
        shift 2
        ;;
    --registry)
        REGISTRY="$2"
        shift 2
        ;;
    --tag|-t)
        IMAGE="$2"
        shift 2
        ;;
    -f|--file)
        DOCKERFILE="$2"
        shift 2
        ;;
    --push)
        PUSH=true
        shift
        ;;
    --cacert)
        CACERT="$2"
        if [ ! -f "${CACERT}" ]; then
            echo "Error: file \"${CACERT}\" does not exist!"
            exit 1
        fi
        shift 2
        ;;
    -*)
        echo "Error: unknown flag '$1'"
        exit 1
        ;;
    *)
        CONTEXT="$1"
        shift
        ;;
    esac
done

if [ -z "$BUILDKIT_NAME" ]; then
    echo 'Error: no buildkit name'
    exit 1
elif [ -z "$IMAGE" ]; then
    echo 'Warning: no tag given'
elif [ -z "$PLATFORMS" ]; then
    echo 'Warning: no platforms'
fi

if [ ! -d "$CONTEXT" ]; then
    echo "Error: build context '$CONTEXT' does not exist"
    exit 1
fi

if [ ! -f "$DOCKERFILE" ]; then
    echo "Error: docker file '$DOCKERFILE' does not exist"
    exit 1
fi

if ! docker buildx use $BUILDKIT_NAME ; then
    CREATE_FLAGS="--use --name $BUILDKIT_NAME --platform $PLATFORMS"
    CREATE_FLAGS="$CREATE_FLAGS --driver-opt network=host"
    if [ -n "$BUILDKIT_CONFIG" ]; then
        if [ ! -f "$BUILDKIT_CONFIG" ]; then
            echo "Warning: no such file '$BUILDKIT_CONFIG'"
        else
            CREATE_FLAGS="$CREATE_FLAGS --config $BUILDKIT_CONFIG"
        fi
    fi
    docker buildx create $CREATE_FLAGS
    docker buildx inspect --bootstrap
    docker run --privileged --rm tonistiigi/binfmt --install all
    if [ -n "${CACERT:-}" -a -f "${CACERT}" ]; then
        container="$(docker buildx inspect "$BUILDKIT_NAME" | awk '/Name:/{print $2}' | grep -Ev "^$BUILDKIT_NAME\$")"
        docker container cp "${CACERT}" "buildx_buildkit_${container}:/usr/local/share/ca-certificates/${BUILDKIT_NAME}.crt"
        docker container exec "buildx_buildkit_${container}" update-ca-certificates --verbose --force
        docker container restart "buildx_buildkit_${container}"
    fi
fi