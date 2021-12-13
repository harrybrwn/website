package main

import (
	"embed"
	"flag"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	port = "8080"
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

	// go :embed templates
	//templates embed.FS
)

func init() {
	flag.StringVar(&port, "port", port, "the port to run the server on")
	flag.Parse()
}

func main() {
	e := echo.New()
	logger := logrus.New()
	db, err := db.Connect()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	e.Use(middleware.Logger())
	e.Use(app.RequestLogRecorder(db, logger))
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			r := c.Request()
			fmt.Println(r.RemoteAddr, r.URL.String(), c.RealIP())
			return next(c)
		}
	})
	e.GET("/", echo.WrapHandler(http.HandlerFunc(harry)))
	e.GET("/pub.asc", echo.WrapHandler(http.HandlerFunc(keys)))
	e.GET("/~harry", echo.WrapHandler(http.HandlerFunc(harry)))
	e.GET("/robots.txt", echo.WrapHandler(http.HandlerFunc(robotsHandler)))

	e.GET("/static/*",
		echo.WrapHandler(staticCache(http.FileServer(http.FS(static)))),
		middleware.Rewrite(map[string]string{"/static/*": "/static/$1"}),
	)

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error {
		return c.JSON(200, app.GetQuotes())
	})
	api.GET("/quote", func(c echo.Context) error {
		return c.JSON(200, app.RandomQuote())
	})

	logger.WithField("time", time.Now()).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
	if err != nil {
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

var startup = time.Now()

func staticCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Last-Modified", startup.Format(time.RFC1123))
		rw.Header().Set("Cache-Control", "public, max-age=31919000")
		h.ServeHTTP(rw, r)
	})
}
