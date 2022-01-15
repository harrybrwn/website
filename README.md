# www.harrybrwn.com

This is the repo for my personal website.

## Build

```
yarn install
yarn build
go generate
go build -o bin/harrybrwn.com
bin/harrybrown.com -env
```

### Run locally

Once the binary is built, you can start the database and run the server in debug
mode.

```
docker-compose up -d db
bin/harrybrwn.com -env -d
```


If you have the heroku cli, you can start the serer with `haroku local web`.

