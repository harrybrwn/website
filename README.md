# www.harrybrwn.com

![ci status](https://github.com/harrybrwn/harrybrwn.com/actions/workflows/ci.yml/badge.svg)

This is the repo for my personal website.

# Prerequisites

- [Docker](https://docs.docker.com/engine/install/)
- Docker's [compose plugin](https://docs.docker.com/compose/install/compose-plugin/)
- [Go](https://go.dev/doc/install) 1.18 or newer
- [node 16.13.x](https://nodejs.org/en/download/) and [yarn 1.22.x](https://classic.yarnpkg.com/lang/en/docs/install/)
- kubectl, helm, minikube

## Build

```
docker compose build
```

### Run locally

Once the binary is built, you can start the database and run the server in debug
mode.

```
./bootstrap.sh
docker compose up
```

If you have the heroku cli, you can start the serer with `haroku local web`.


## Tests

### Backend Unit Tests

```
go test -tags ci ./...
```

### Backend Functional Tests

```
docker-compose -f docker-compose.test.yml build
docker-compose -f docker-compose.yml -f docker-compose.test.yml up -d db redis web
docker-compose -f docker-compose.test.yml run --rm tests scripts/functional-tests.sh
```

### Frontend

```
yarn test
```

## Deployment

```
docker context use harrybrwn # send docker commands to prod box
docker network rm ingress
docker network create --driver overlay --ingress --scope swarm --ipv6 harrybrwn-net
# make should all worker nodes are connected at this point
docker node ls
docker stack rm harrybrwn
# Build
env $(cat .env) docker buildx bake \
  -f docker-compose.yml \
  -f config/docker/buildx.yml \
  --push
docker-compose \
    -f docker-compose.yml \
    -f config/docker/prod.yml config | \
  docker stack deploy \
   --resolve-image always \
   --with-registry-auth \
   --prune \
   -c - \
   harrybrwn
docker service ls
docker service logs -f harrybrwn_nginx
```

