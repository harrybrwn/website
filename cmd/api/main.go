package main

import (
	_ "embed"

	flag "github.com/spf13/pflag"

	"harrybrown.com/pkg/log"
)

var (
	//go:embed pub.asc
	gpgPubkey []byte

	logger = log.SetLogger(log.New(log.WithEnv(), log.WithServiceName("api")))
)

func main() {
	var (
		port = 8081
	)
	_ = gpgPubkey
	flag.IntVarP(&port, "port", "p", port, "server port")
	flag.Parse()
	logger.Infof("starting server on [::]:%d", port)
}
