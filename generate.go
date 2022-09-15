//go:build generate
// +build generate

package main

//go:generate echo Generating build files...

/// Generate mock stubs
//go:generate sh scripts/mockgen.sh

/// Files needed by the api
//go :generate cp build/harrybrwn.com/invite/index.html cmd/api/invite.html
//go:generate cp frontend/public/pub.asc cmd/api/
