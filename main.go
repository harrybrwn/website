package main

import (
	"embed"
	"encoding/hex"
	"flag"
	"io/fs"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

//go:generate sh scripts/mockgen.sh

const buildDir = "./build"

var (
	//go:embed build/index.html
	harryStaticPage []byte
	//go:embed build/remora/index.html
	remoraStaticPage []byte
	//go:embed build/harry_y_tanya/index.html
	harryYTanyaStaticPage []byte
	//go:embed build/404.html
	notFoundStaticPage []byte
	//go:embed build/admin/index.html
	adminPageStatic []byte
	//go:embed build/games/index.html
	gamesStaticPage []byte
	//TODO go:embed build/tanya/index.html
	//tanyaStaticPage []byte

	//go:embed files/bookmarks.json
	bookmarks []byte
	//go:embed build/pub.asc
	gpgPubkey []byte
	//go:embed build/robots.txt
	robots []byte
	//go:embed build/favicon.ico
	favicon []byte
	//go:embed build/manifest.json
	manifest []byte
	//go:embed build/static
	static embed.FS
	//go:embed build/sitemap.xml
	sitemap []byte
	// go :embed build/sitemap.xml.gz
	sitemapgz []byte

	//go:embed frontend/templates
	templates embed.FS

	logger = logrus.New()
)

func main() {
	var (
		port = "8080"
		env  bool
		e    = echo.New()
	)
	flag.StringVar(&port, "port", port, "the port to run the server on")
	flag.BoolVar(&env, "env", env, "read .env")
	flag.Parse()

	app.SetLogger(logger)
	e.Logger = log.WrapLogrus(logger)
	e.Debug = app.Debug
	e.DisableHTTP2 = false
	e.HideBanner = true

	if env {
		godotenv.Load()
	}

	if app.Debug {
		godotenv.Load()
		auth.Expiration = time.Hour * 24
		auth.RefreshExpiration = auth.Expiration * 2
		logger.SetLevel(logrus.DebugLevel)
	}

	echo.NotFoundHandler = NotFoundHandler()

	db, err := db.Connect(logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()

	templates, err := fs.Sub(templates, "frontend")
	if err != nil {
		logger.Fatal(err)
	}

	jwtConf := NewTokenConfig()
	guard := auth.Guard(jwtConf)
	e.Pre(app.RequestLogRecorder(db, logger))

	e.GET("/", page(harryStaticPage, buildDir+"/index.html"))
	e.GET("/~harry", page(harryStaticPage, buildDir+"/index.html"))
	e.GET("/tanya/hyt", page(harryYTanyaStaticPage, buildDir+"/harry_y_tanya/index.html"), guard)
	e.GET("/remora", page(remoraStaticPage, buildDir+"/remora/index.html"))
	e.GET("/games", page(gamesStaticPage, buildDir+"/games/index.html"), guard)
	e.GET("/admin", page(adminPageStatic, buildDir+"/admin/index.html"), guard, auth.AdminOnly())
	e.GET("/old", echo.WrapHandler(app.HomepageHandler(templates)), guard)

	e.GET("/static/*", echo.WrapHandler(handleStatic()))
	e.GET("/pub.asc", WrapHandler(keys))
	e.GET("/robots.txt", WrapHandler(robotsHandler))
	e.GET("/sitemap.xml", WrapHandler(sitemapHandler))
	e.GET("/sitemap.xml.gz", WrapHandler(sitemapGZHandler))
	e.GET("/favicon.ico", faviconHandler())
	e.GET("/manifest.json", json(manifest))
	e.GET("/secret", func(c echo.Context) error {
		return c.HTML(200, "<h1>This is a secret</h1>")
	}, guard, auth.AdminOnly())

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error { return c.JSON(200, app.GetQuotes()) })
	api.GET("/quote", func(c echo.Context) error { return c.JSON(200, app.RandomQuote()) })
	api.GET("/bookmarks", json(bookmarks))
	api.GET("/hits", app.Hits(db))
	api.POST("/token", app.TokenHandler(jwtConf, app.NewUserStore(db)))
	api.POST("/refresh", app.RefreshTokenHandler(jwtConf), guard)
	api.GET("/runtime", func(c echo.Context) error {
		return c.JSON(200, app.RuntimeInfo(startup))
	}, guard, auth.AdminOnly())
	api.Any("/ping", WrapHandler(ping))
	api.GET("/logs", app.LogListHandler(db))

	logger.WithField("time", startup).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
	if err != nil {
		logger.Fatal(err)
	}
}

const (
	tokenKey     = "_token"
	maxCookieAge = 2147483647
)

func NewTokenConfig() auth.TokenConfig {
	hexseed, hasSeed := os.LookupEnv("JWT_SEED")
	if hasSeed {
		logger.Info("creating token config from seed")
		seed, err := hex.DecodeString(hexseed)
		if err != nil {
			panic(errors.Wrap(err, "could not decode private key seed from hex"))
		}
		return &tokenConfig{auth.EdDSATokenConfigFromSeed(seed)}
	}
	logger.Warn("generating new key pair for token config")
	return &tokenConfig{auth.GenEdDSATokenConfig()}
}

type tokenConfig struct{ auth.TokenConfig }

func (tc *tokenConfig) GetToken(r *http.Request) (string, error) {
	c, err := r.Cookie(tokenKey)
	if err != nil {
		return auth.GetBearerToken(r)
	}
	return c.Value, nil
}

func NotFoundHandler() echo.HandlerFunc {
	if app.Debug {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Request().RequestURI, "/api") {
				return echo.ErrNotFound
			}
			return serveFile(c, "build/pages/404.html")
		}
	}
	return func(c echo.Context) error {
		if strings.HasPrefix(c.Request().RequestURI, "/api") {
			return echo.ErrNotFound
		}
		return c.HTMLBlob(404, notFoundStaticPage)
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	staticLastModified(rw.Header())
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(gpgPubkey)
}

func faviconHandler() echo.HandlerFunc {
	length := strconv.FormatInt(int64(len(favicon)), 10)
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("Content-Length", length)
		h.Set("Accept-Ranges", "bytes")
		h.Set("Cache-Control", "public, max-age=31919000")
		staticLastModified(h)
		return c.Blob(200, "image/x-icon", favicon)
	}
}

func page(raw []byte, filename string) echo.HandlerFunc {
	var (
		hf     echo.HandlerFunc
		length = int64(len(raw))
	)
	if app.Debug {
		hf = func(c echo.Context) error {
			return serveFile(c, filename)
		}
		b, err := os.ReadFile(filename)
		if err != nil {
			panic(err)
		}
		if http.DetectContentType(b) == "application/x-gzip" {
			hf = gzip(hf)
		}
	} else {
		ct := http.DetectContentType(raw)
		hf = func(c echo.Context) error {
			h := c.Response().Header()
			staticLastModified(h)
			h.Set("Cache-Control", "public, max-age=31919000")
			h.Set("Content-Length", strconv.FormatInt(length, 10))
			h.Set("Accept-Ranges", "bytes")
			return c.Blob(200, ct, raw)
		}
		if http.DetectContentType(raw) == "application/x-gzip" {
			hf = gzip(hf)
		}
	}
	return hf
}

func json(raw []byte) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		staticLastModified(h)
		h.Set("Cache-Control", "public, max-age=31919000")
		h.Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
		return c.Blob(200, "application/json", raw)
	}
}

func serveFile(c echo.Context, filename string) error {
	http.ServeFile(c.Response(), c.Request(), filename)
	return nil
}

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	staticLastModified(rw.Header())
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(robots)
}

func sitemapHandler(rw http.ResponseWriter, r *http.Request) {
	h := rw.Header()
	staticLastModified(h)
	h.Set("Cache-Control", "public, max-age=31919000")
	h.Set("Content-Length", strconv.FormatInt(int64(len(sitemap)), 10))
	h.Set("Content-Type", "text/xml")
	rw.Write(sitemap)
}

func sitemapGZHandler(rw http.ResponseWriter, r *http.Request) {
	h := rw.Header()
	staticLastModified(h)
	h.Set("Cache-Control", "public, max-age=31919000")
	h.Set("Content-Length", strconv.FormatInt(int64(len(sitemapgz)), 10))
	h.Set("Content-Encoding", "gzip")
	h.Set("Content-Type", "text/xml")
	rw.Write(sitemapgz)
}

func handleStatic() http.Handler {
	if app.Debug {
		h := http.StripPrefix("/static/", http.FileServer(http.Dir("build/static")))
		return h
	}
	fs, err := fs.Sub(static, "build")
	if err != nil {
		fs = static
	}
	return staticCache(http.FileServer(http.FS(fs)))
}

func ping(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(200)
}

func gzip(handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		accept := c.Request().Header.Get("Accept-Encoding")
		if !strings.Contains(accept, "gzip") {
			logger.WithField("accept-encoding", accept).Error("browser encoding not supported")
			return c.Blob(500, "text/html", []byte("<h2>encoding failure</h2>"))
		}
		c.Response().Header().Set("Content-Encoding", "gzip")
		return handler(c)
	}
}

var startup = time.Now()

func staticLastModified(h http.Header) {
	h.Set("Last-Modified", startup.UTC().Format(http.TimeFormat))
}

func staticCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		header := rw.Header()
		staticLastModified(header)
		header.Set("Cache-Control", "public, max-age=31919000")
		h.ServeHTTP(rw, r)
	})
}

func WrapHandler(h http.HandlerFunc) echo.HandlerFunc {
	return echo.WrapHandler(h)
}
