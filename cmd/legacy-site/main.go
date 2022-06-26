package main

import (
	"os"

	"github.com/go-chi/chi/v5"
	flag "github.com/spf13/pflag"
	"harrybrown.com/app"
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
	flag.StringVar(&templateDir, "templates", templateDir, "directory containing template files")
	flag.Parse()

	templates := os.DirFS(templateDir)

	jwtConf := app.NewTokenConfig()
	r := chi.NewRouter()
	g := auth.Guard(jwtConf)
	r.With(g).Get("/old", app.OldHomepageHandler(templates).ServeHTTP)
	if err := web.ListenAndServe(":8083", r); err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}
