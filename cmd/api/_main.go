package main

/*

import (
	_ "embed"
	"fmt"

	"github.com/go-chi/chi/v5"
	flag "github.com/spf13/pflag"

	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	logger = log.SetLogger(log.New(log.WithEnv(), log.WithServiceName("api")))
)

func main() {
	var (
		port = 8086
	)
	flag.IntVarP(&port, "port", "p", port, "server port")
	flag.Parse()
	logger.Infof("starting server on [::]:%d", port)

	r := chi.NewRouter()
	r.Use(
		web.AccessLog(logger),
		web.Metrics(),
	)
	err := web.ListenAndServe(fmt.Sprintf(":%d", port), r)
	if err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}
*/
