# www.harrybrwn.com

This is the repo for my personal website.

## Build

```
yarn install
yarn build
go generate
go build -o bin/harrybrwn.com
```

### Run locally

Once the binary is built, you can start the database and run the server in debug
mode.

```
docker-compose up -d db redis
bin/harrybrwn.com -env
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
