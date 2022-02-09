#!/bin/sh

set -e

dest="internal/mocks"

mockgen                             \
	-package mockdb                 \
	-destination $dest/mockdb/db.go \
	harrybrown.com/pkg/db DB,Rows

mockgen                                 \
	-package mockapp                    \
	-destination $dest/mockapp/redis.go \
	harrybrown.com/app HitsCache
