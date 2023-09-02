#!/bin/sh

set -eu

dest="pkg/internal/mocks"
pkg=gopkg.hrry.dev/homelab

if [ -n "${NIX_BUILD_CORES:-}" ]; then
	set -x
fi


mockgen -package mockdb -destination $dest/mockdb/db.go "$pkg/pkg/db" DB,Rows
mockgen -package mockredis -destination $dest/mockredis/cmdable.go github.com/go-redis/redis/v8 Cmdable,UniversalClient
mockgen -package mockusers -destination $dest/mockusers/store.go "$pkg/pkg/app" UserStore
mockgen -package mockrows -destination $dest/mockrows/db.go "$pkg/pkg/db" Rows
mockgen -package mockws -destination $dest/mockws/ws.go "$pkg/pkg/ws" Connection
mockgen -package mockemail -destination $dest/mockemail/email.go "$pkg/pkg/email" Client
mockgen -package mockinvite -destination $dest/mockinvite/invites.go "$pkg/pkg/invite" Store,Mailer

