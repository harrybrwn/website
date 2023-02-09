# Helm Charts and Repo

To update the helm repo, update the index file then run

```
make deploy
```

## Setup R2

The makefile uses [Minio's `mc`](https://min.io/download#/linux).

```
mc alias set r2 https://<account>.r2.cloudflarestorage.com <access-key> <secret-key>
```
