package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	flag "github.com/spf13/pflag"
	"harrybrown.com/pkg/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var logger = log.SetLogger(log.New(
	log.WithEnv(),
	log.WithServiceName("legacy-site"),
))

func main() {
	templateDir := "frontend/templates"
	port := 8083
	flag.StringVar(&templateDir, "templates", templateDir, "directory containing template files")
	flag.IntVarP(&port, "port", "p", port, "port to run the server on")
	flag.Parse()

	templates := os.DirFS(templateDir)

	jwtConf := app.NewTokenConfig()
	r := chi.NewRouter()
	g := auth.Guard(jwtConf)
	r.With(g).Get("/old", app.OldHomepageHandler(templates).ServeHTTP)
	r.Head("/health/ready", func(w http.ResponseWriter, r *http.Request) {})
	if err := web.ListenAndServe(fmt.Sprintf(":%d", port), r); err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}
