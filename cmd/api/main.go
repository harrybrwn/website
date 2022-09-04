package main

import (
	_ "embed"

	flag "github.com/spf13/pflag"

	"harrybrown.com/pkg/log"
)

var (
	//go:embed invite.html
	inviteStaticPage []byte
	//go:embed invite_email.html
	inviteEmailStatic []byte
	//go:embed bookmarks.json
	bookmarks []byte
	//go:embed pub.asc
	gpgPubkey []byte

	logger = log.SetLogger(log.New(log.WithEnv(), log.WithServiceName("api")))
)

func main() {
	var (
		port = 8081
	)
	flag.IntVarP(&port, "port", "p", port, "server port")
	flag.Parse()
}
