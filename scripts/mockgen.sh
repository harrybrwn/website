#!/bin/sh

set -eu

dest="internal/mocks"

if [ -n "${NIX_BUILD_CORES:-}" ]; then
	set -x
fi

mockgen -package mockdb -destination $dest/mockdb/db.go harrybrown.com/pkg/db DB,Rows
mockgen -package mockredis -destination $dest/mockredis/cmdable.go github.com/go-redis/redis/v8 Cmdable,UniversalClient
mockgen -package mockusers -destination $dest/mockusers/store.go harrybrown.com/app UserStore
mockgen -package mockrows -destination $dest/mockrows/db.go harrybrown.com/pkg/db Rows
mockgen -package mockws -destination $dest/mockws/ws.go harrybrown.com/pkg/ws Connection
mockgen -package mockemail -destination $dest/mockemail/email.go harrybrown.com/pkg/email Client
mockgen -package mockinvite -destination $dest/mockinvite/invites.go harrybrown.com/pkg/invite Store,Mailer
