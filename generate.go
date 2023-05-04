//go:build generate
// +build generate

package main

import (
	_ "github.com/golang/mock/gomock"
	_ "github.com/golang/mock/mockgen"
	_ "golang.org/x/tools/go/packages"
)

//go:generate echo Generating build files...

/// Generate mock stubs
//go:generate scripts/mockgen.sh

/// Files needed by the api
//go :generate cp build/harrybrwn.com/invite/index.html cmd/api/invite.html
//go:generate cp frontend/legacy/public/pub.asc cmd/api/
