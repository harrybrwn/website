package main

import (
	"flag"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/log"
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
	addr := ":8083"
	logger.WithField("address", addr).Info("starting server")
	http.ListenAndServe(addr, r)
}
