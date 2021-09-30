package main

import (
	"embed"
	"net/http"
	"os"

	"harrybrown.com/app"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	mux    = http.NewServeMux()
	router = web.NewRouter()
	port   = "8080"
)

var (
	//go:embed embeds/harry.html
	harryStaticPage []byte
	//go:embed embeds/keys/pub.asc
	pubkey []byte
	//go:embed embeds/robots.txt
	robots []byte
	//go:embed static/css static/data static/files static/img static/js
	static embed.FS

	//go :embed templates
	//templates embed.FS
)

func init() {
	app.StringFlag(&port, "port", "the port to run the server on")
	app.ParseFlags()

	router.SetMux(mux)
	router.HandleRoutes(app.Routes)
}

func main() {
	if app.Debug {
		log.Printf("running on localhost:%s\n", port)
		router.AddRoute("/static/", app.NewFileServer("static")) // handle file server
	} else {
		router.AddRoute("/static/", http.FileServer(http.FS(static)))
	}

	mux.HandleFunc("/~harry", harry)
	mux.HandleFunc("/robots.txt", robotsHandler)
	mux.HandleFunc("/pub.asc", keys)

	handler := logger(log.NewPlainLogger(os.Stdout), mux)
	server := http.Server{
		Addr:    ":" + port,
		Handler: handler,
	}
	if router.HandlerHook != nil {
		server.Handler = router.HandlerHook(handler)
	}
	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	rw.Write(pubkey)
}

func harry(wr http.ResponseWriter, r *http.Request) {
	wr.Header().Set("Content-Type", "text/html")
	wr.Write(harryStaticPage)
}

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Write(robots)
}

func logger(logger log.PrintLogger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		country := r.Header.Get("CF-IPCountry")
		logger.Printf(
			"[%s] %s country=%s\n",
			r.Method, r.RequestURI,
			country,
		)
		h.ServeHTTP(rw, r)
	})
}
