package main

import (
	"embed"
	"net/http"
	"os"
	"time"

	"harrybrown.com/app"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	mux    = http.NewServeMux()
	router = web.NewRouter()
	port   = "8080"

	built string
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
	//go:embed templates
	templates embed.FS
)

func init() {
	app.StringFlag(&port, "port", "the port to run the server on")
	app.ParseFlags()
	web.DefaultErrorHandler = app.NotFoundHandler(templates)

	router.SetMux(mux)
}

func main() {
	if app.Debug {
		log.Printf("running on localhost:%s\n", port)
		mux.Handle("/static/", app.NewFileServer("static"))
	} else {
		mux.Handle("/static/", staticCache(http.FileServer(http.FS(static))))
	}

	mux.HandleFunc("/~harry", harry)
	mux.HandleFunc("/robots.txt", robotsHandler)
	mux.HandleFunc("/pub.asc", keys)
	mux.HandleFunc("/", app.HomepageHandler(templates))
	mux.Handle("/api/info", web.APIHandler(app.HandleInfo))
	mux.Handle("/api/quotes", web.APIHandler(func(rw http.ResponseWriter, r *http.Request) interface{} {
		return app.GetQuotes()
	}))
	mux.Handle("/api/quote", web.APIHandler(func(rw http.ResponseWriter, r *http.Request) interface{} {
		return app.RandomQuote()
	}))

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
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(pubkey)
}

func harry(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Content-Type", "text/html")
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(harryStaticPage)
}

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
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

func staticCache(h http.Handler) http.Handler {
	t, err := time.Parse(time.RFC1123, built)
	if err != nil {
		panic(err)
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Last-Modified", t.Format(time.RFC1123))
		rw.Header().Set("Cache-Control", "public, max-age=31919000")
		h.ServeHTTP(rw, r)
	})
}
