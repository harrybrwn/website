#!/bin/sh

set -e

dest="internal/mocks"

mockgen                             \
	-package mockdb                 \
	-destination $dest/mockdb/db.go \
	harrybrown.com/pkg/db DB,Rows

mockgen -package mockredis -destination $dest/mockredis/cmdable.go \
	github.com/go-redis/redis/v8 Cmdable

mockgen -package mockusers -destination $dest/mockusers/store.go \
	harrybrown.com/app UserStore

mockgen -package mocksendgrid -destination $dest/mocksendgrid/client.go \
  harrybrwn.com/app EmailClient
