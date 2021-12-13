package main

import (
	"embed"
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	//go:embed embeds/harry.html
	harryStaticPage []byte
	//go:embed embeds/pub.asc
	pubkey []byte
	//go:embed embeds/robots.txt
	robots []byte
	//go:embed static/css static/data static/files static/img static/js
	static embed.FS
	//go:embed embeds/favicon.ico
	favicon []byte

	// go :embed templates
	//templates embed.FS
)

func main() {
	var (
		port   = "8080"
		e      = echo.New()
		logger = logrus.New()
	)
	flag.StringVar(&port, "port", port, "the port to run the server on")
	flag.Parse()

	e.HideBanner = true
	db, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	e.Use(app.RequestLogRecorder(db, logger))
	e.GET("/", echo.WrapHandler(harry()))
	e.GET("/pub.asc", echo.WrapHandler(http.HandlerFunc(keys)))
	e.GET("/~harry", echo.WrapHandler(harry()))
	e.GET("/robots.txt", echo.WrapHandler(http.HandlerFunc(robotsHandler)))
	e.GET("/favicon.ico", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=31919000")
		return c.Blob(200, "image/x-icon", favicon)
	})
	e.GET("/static/*", echo.WrapHandler(handleStatic()))

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error {
		return c.JSON(200, app.GetQuotes())
	})
	api.GET("/quote", func(c echo.Context) error {
		return c.JSON(200, app.RandomQuote())
	})

	logger.WithField("time", startup).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
	if err != nil {
		log.Fatal(err)
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(pubkey)
}

func harry() http.Handler {
	if app.Debug {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Set("Content-Type", "text/html")
			rw.Header().Set("Cache-Control", "public, max-age=31919000")
			http.ServeFile(rw, r, "embeds/harry.html")
		})
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "text/html")
		rw.Header().Set("Cache-Control", "public, max-age=31919000")
		rw.Write(harryStaticPage)
	})
}

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(robots)
}

func handleStatic() http.Handler {
	return staticCache(http.FileServer(http.FS(static)))
}

var startup = time.Now()

func staticCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Last-Modified", startup.Format(time.RFC1123))
		rw.Header().Set("Cache-Control", "public, max-age=31919000")
		h.ServeHTTP(rw, r)
	})
}
