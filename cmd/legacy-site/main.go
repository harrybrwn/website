package main

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var logger = log.GetLogger()

func main() {
	jwtConf := app.NewTokenConfig()
	r := chi.NewRouter()
	r.With(auth.Guard(jwtConf)).Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello! :)"))
	})
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		web.WriteError(w, &web.Error{Message: "test error", Status: 418})
	})
	addr := ":8083"
	logger.WithField("address", addr).Info("starting server")
	http.ListenAndServe(addr, r)
}
