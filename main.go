package main

import (
	"embed"
	"encoding/hex"
	"flag"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
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
	hytStaticPage []byte
	//go:embed build/404.html
	notFoundStaticPage []byte
	//go:embed build/admin/index.html
	adminStaticPage []byte
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
		auth.Expiration = time.Second * 30
		// auth.RefreshExpiration = auth.Expiration * 60
		logger.SetLevel(logrus.DebugLevel)
	}

	echo.NotFoundHandler = NotFoundHandler()

	db, rd, err := db.Datastores(logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	defer rd.Close()

	templates, err := fs.Sub(templates, "frontend")
	if err != nil {
		logger.Fatal(err)
	}

	jwtConf := NewTokenConfig()
	guard := auth.Guard(jwtConf)
	e.Pre(app.RequestLogRecorder(db, logger))

	e.GET("/", page(harryStaticPage, "index.html"))
	e.GET("/~harry", page(harryStaticPage, "index.html"))
	e.GET("/tanya/hyt", page(hytStaticPage, "harry_y_tanya/index.html"), guard)
	e.GET("/remora", page(remoraStaticPage, "remora/index.html"))
	e.GET("/games", page(gamesStaticPage, "games/index.html"), guard)
	e.GET("/admin", page(adminStaticPage, "admin/index.html"), guard, auth.AdminOnly())
	e.GET("/old", echo.WrapHandler(app.HomepageHandler(templates)), guard)

	e.GET("/static/*", echo.WrapHandler(handleStatic()))
	e.GET("/pub.asc", WrapHandler(keys))
	e.GET("/robots.txt", WrapHandler(robotsHandler))
	e.GET("/sitemap.xml", WrapHandler(sitemapHandler(sitemap, false)))
	e.GET("/sitemap.xml.gz", WrapHandler(sitemapHandler(sitemapgz, true)))
	e.GET("/favicon.ico", faviconHandler())
	e.GET("/manifest.json", json(manifest))
	e.GET("/secret", func(c echo.Context) error {
		return c.HTML(200, "<h1>This is a secret</h1>")
	}, guard, auth.AdminOnly())

	api := e.Group("/api")
	tokenSrv := app.TokenService{
		Config: jwtConf,
		Tokens: auth.NewRedisTokenStore(auth.RefreshExpiration, rd),
		Users:  app.NewUserStore(db),
	}
	api.POST("/token", tokenSrv.Token)
	api.POST("/refresh", tokenSrv.Refresh)
	api.POST("/revoke", tokenSrv.Revoke, guard)
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error { return c.JSON(200, app.GetQuotes()) })
	api.GET("/quote", func(c echo.Context) error { return c.JSON(200, app.RandomQuote()) })
	api.GET("/bookmarks", json(bookmarks))
	api.GET("/hits", app.Hits(db))
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
			return serveFile(c, 404, "build/pages/404.html")
		}
	}
	return func(c echo.Context) error {
		if strings.HasPrefix(c.Request().RequestURI, "/api") {
			return echo.ErrNotFound
		}
		return c.HTMLBlob(404, notFoundStaticPage)
	}
}

const staticCacheControl = "public, max-age=31919000"

func keys(rw http.ResponseWriter, r *http.Request) {
	staticLastModified(rw.Header())
	rw.Header().Set("Cache-Control", staticCacheControl)
	rw.Write(gpgPubkey)
}

func faviconHandler() echo.HandlerFunc {
	length := strconv.FormatInt(int64(len(favicon)), 10)
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("Content-Length", length)
		h.Set("Accept-Ranges", "bytes")
		h.Set("Cache-Control", staticCacheControl)
		staticLastModified(h)
		return c.Blob(200, "image/x-icon", favicon)
	}
}

func page(raw []byte, filename string) echo.HandlerFunc {
	var (
		hf     echo.HandlerFunc
		length = int64(len(raw))
	)
	filename = filepath.Join(buildDir, filename)
	if app.Debug {
		hf = func(c echo.Context) error {
			return serveFile(c, 200, filename)
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
			h.Set("Cache-Control", staticCacheControl)
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
		h.Set("Cache-Control", staticCacheControl)
		h.Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
		return c.Blob(200, "application/json", raw)
	}
}

func serveFile(c echo.Context, status int, filename string) error {
	http.ServeFile(c.Response(), c.Request(), filename)
	c.Response().WriteHeader(status)
	return nil
}

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	h := rw.Header()
	staticLastModified(h)
	h.Set("Cache-Control", staticCacheControl)
	h.Set("Content-Type", "text/plain")
	rw.Write(robots)
}

func sitemapHandler(raw []byte, gzip bool) func(http.ResponseWriter, *http.Request) {
	length := strconv.FormatInt(int64(len(raw)), 10)
	return func(rw http.ResponseWriter, r *http.Request) {
		h := rw.Header()
		staticLastModified(h)
		h.Set("Cache-Control", staticCacheControl)
		h.Set("Content-Length", length)
		h.Set("Content-Type", "text/xml")
		if gzip {
			h.Set("Content-Encoding", "gzip")
		}
		rw.Write(raw)
	}
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

func acceptsGzip(header http.Header) bool {
	accept := header.Get("Accept-Encoding")
	return strings.Contains(accept, "gzip")
}

func gzip(handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !acceptsGzip(c.Request().Header) {
			logger.WithField(
				"accept-encoding", c.Request().Header.Get("Accept-Encoding"),
			).Error("browser encoding not supported")
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
		header.Set("Cache-Control", staticCacheControl)
		h.ServeHTTP(rw, r)
	})
}

func WrapHandler(h http.HandlerFunc) echo.HandlerFunc {
	return echo.WrapHandler(h)
}
