package main

import (
	"database/sql"
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

var (
	//go:embed build/index.html
	harryStaticPage []byte
	//go:embed build/pages/remora.html
	remoraStaticPage []byte
	//go:embed build/pages/harry-y-tanya.html
	harryYTanyaStaticPage []byte
	//go:embed build/pub.asc
	gpgPubkey []byte
	//go:embed build/robots.txt
	robots []byte
	//go:embed build/favicon.ico
	favicon []byte
	//go:embed build/static
	static embed.FS
	//go:embed build/sitemap.xml
	sitemap []byte
	// go :embed build/sitemap.xml.gz
	sitemapgz []byte

	//go:embed templates
	templates embed.FS

	assetsGziped bool
	logger       = logrus.New()
)

func main() {
	var (
		port = "8080"
		e    = echo.New()
		env  bool
	)
	e.Logger = log.WrapLogrus(logger)
	flag.StringVar(&port, "port", port, "the port to run the server on")
	flag.BoolVar(&env, "env", env, "read environment files from .env")
	flag.BoolVar(&assetsGziped, "gzip", assetsGziped, "use this flag when all assets have been gzip ahead of time")
	flag.Parse()

	if env {
		godotenv.Load()
	}
	if app.Debug {
		auth.Expiration = time.Hour * 24
		auth.RefreshExpiration = auth.Expiration * 2
	}

	e.HideBanner = true
	db, err := db.Connect(logger)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	jwtConf := NewTokenConfig()
	guard := auth.Guard(jwtConf)
	e.Use(app.RequestLogRecorder(db, logger))

	e.GET("/", echo.WrapHandler(index()))
	e.GET("/~harry", echo.WrapHandler(index()))
	e.GET("/remora", remoraPage(), staticMiddleware)
	e.GET("/tanya/hyt", echo.WrapHandler(page(harryYTanyaStaticPage, "build/pages/harry-y-tanya.html")), guard)
	e.GET("/~tanya", func(c echo.Context) error {
		return c.HTML(http.StatusNotImplemented, "<p>This is not finished yet</p>")
	}, guard)

	e.GET("/static/*", echo.WrapHandler(handleStatic()))
	e.GET("/pub.asc", echo.WrapHandler(http.HandlerFunc(keys)))
	e.GET("/robots.txt", echo.WrapHandler(http.HandlerFunc(robotsHandler)))
	e.GET("/sitemap.xml", echo.WrapHandler(http.HandlerFunc(sitemapHandler)))
	e.GET("/sitemap.xml.gz", echo.WrapHandler(http.HandlerFunc(sitemapGZHandler)))
	e.GET("/favicon.ico", func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("Content-Length", strconv.FormatInt(int64(len(favicon)), 10))
		h.Set("Accept-Ranges", "bytes")
		return c.Blob(200, "image/x-icon", favicon)
	}, staticMiddleware)
	e.GET("/old", echo.WrapHandler(app.HomepageHandler(templates)), guard)
	e.GET("/secret", func(c echo.Context) error {
		return c.HTML(200, `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta http-equiv="X-UA-Compatible" content="IE=edge">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>Secret</title>
</head>
<body>
	<h1>This is a Secret</h1>
</body>
</html>`)
	}, guard, auth.AdminOnly())

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error {
		return c.JSON(200, app.GetQuotes())
	})
	api.GET("/quote", func(c echo.Context) error {
		return c.JSON(200, app.RandomQuote())
	})
	api.GET("/hits", hits(db))
	api.POST("/token", TokenHandler(jwtConf, app.NewUserStore(db)))
	api.GET("/runtime", func(c echo.Context) error {
		return c.JSON(200, map[string]interface{}{
			"debug":    app.Debug,
			"birthday": app.GetBirthday(),
		})
	}, guard, auth.AdminOnly())
	api.Any("/ping", echo.WrapHandler(http.HandlerFunc(ping)))

	logger.WithField("time", startup).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
	if err != nil {
		log.Fatal(err)
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

func TokenHandler(conf auth.TokenConfig, store app.UserStore) echo.HandlerFunc {
	return func(c echo.Context) error {
		var (
			err         error
			body        app.Login
			req         = c.Request()
			ctx         = req.Context()
			cookieQuery = req.URL.Query().Get("cookie")
			setCookie   bool
		)
		switch err = c.Bind(&body); err {
		case nil:
			break
		case echo.ErrUnsupportedMediaType:
			return err
		default:
			err = errors.Wrap(err, "failed to bind user data")
			return wrap(err, http.StatusInternalServerError)
		}
		if len(cookieQuery) > 0 {
			setCookie, err = strconv.ParseBool(cookieQuery)
			if err != nil {
				return &echo.HTTPError{Code: http.StatusBadRequest}
			}
		}
		logger.WithFields(logrus.Fields{
			"username": body.Username,
			"email":    body.Email,
		}).Info("getting token")
		if len(body.Password) == 0 {
			return failure(http.StatusBadRequest, "user gave zero length password")
		}
		u, err := store.Login(ctx, &body)
		if err != nil {
			return wrap(err, 404, "could not find user")
		}
		resp, err := auth.NewTokenResponse(conf, u.NewClaims())
		if err != nil {
			err = errors.Wrap(err, "could not create token response")
			return wrap(err, http.StatusInternalServerError)
		}
		if setCookie {
			c.SetCookie(&http.Cookie{
				Name:    tokenKey,
				Value:   resp.Token,
				Expires: time.Unix(resp.Expires, 0),
				Path:    "/",
			})
		}
		return c.JSON(200, resp)
	}
}

func hits(db *sql.DB) echo.HandlerFunc {
	const query = `select count(*) from request_log where uri = $1`
	return func(c echo.Context) error {
		var (
			n int
			u = c.QueryParam("u")
		)
		if len(u) == 0 {
			u = "/"
		}
		row := db.QueryRowContext(c.Request().Context(), query, u)
		if err := row.Scan(&n); err != nil {
			return echo.ErrInternalServerError
		}
		return c.JSON(200, map[string]int{"count": n})
	}
}

func wrap(err error, status int, message ...string) error {
	var msg string
	if len(message) < 1 {
		msg = http.StatusText(status)
	} else {
		msg = message[0]
	}
	return &echo.HTTPError{
		Code:     status,
		Message:  msg,
		Internal: err,
	}
}

func failure(status int, internal string) error {
	return &echo.HTTPError{
		Code:     status,
		Message:  http.StatusText(status),
		Internal: errors.New(internal),
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	staticLastModified(rw.Header())
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(gpgPubkey)
}

func index() http.Handler {
	var hf http.Handler
	if app.Debug {
		hf = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Set("Content-Type", "text/html")
			rw.Header().Set("Cache-Control", "public, max-age=31919000")
			http.ServeFile(rw, r, "build/index.html")
		})
	} else {
		hf = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			h := rw.Header()
			staticLastModified(h)
			h.Set("Content-Type", "text/html")
			h.Set("Cache-Control", "public, max-age=31919000")
			h.Set("Content-Length", strconv.FormatInt(int64(len(harryStaticPage)), 10))
			h.Set("Accept-Ranges", "bytes")
			rw.Write(harryStaticPage)
		})
		if http.DetectContentType(harryStaticPage) == "application/x-gzip" {
			hf = wrapAsGzip(hf)
		}
	}
	return hf
}

func page(raw []byte, filename string) http.Handler {
	var (
		hf     http.Handler
		length = int64(len(raw))
	)
	if app.Debug {
		hf = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Set("Content-Type", "text/html")
			rw.Header().Set("Cache-Control", "public, max-age=31919000")
			http.ServeFile(rw, r, filename)
		})
	} else {
		hf = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			h := rw.Header()
			staticLastModified(h)
			h.Set("Content-Type", "text/html")
			h.Set("Cache-Control", "public, max-age=31919000")
			h.Set("Content-Length", strconv.FormatInt(length, 10))
			h.Set("Accept-Ranges", "bytes")
			rw.Write(harryStaticPage)
		})
		if http.DetectContentType(harryStaticPage) == "application/x-gzip" {
			hf = wrapAsGzip(hf)
		}
	}
	return hf
}

func remoraPage() echo.HandlerFunc {
	if app.Debug {
		return func(c echo.Context) error {
			return c.File("./build/pages/remora.html")
		}
	}
	return func(c echo.Context) error {
		return c.HTMLBlob(200, remoraStaticPage)
	}
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
		if assetsGziped {
			return wrapAsGzip(h)
		}
		return h
	}
	fs, err := fs.Sub(static, "build")
	if err != nil {
		fs = static
	}
	if assetsGziped {
		return wrapAsGzip(staticCache(http.FileServer(http.FS(fs))))
	}
	return staticCache(http.FileServer(http.FS(fs)))
}

func ping(rw http.ResponseWriter, r *http.Request) {
	rw.WriteHeader(200)
}

func wrapAsGzip(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		accept := r.Header.Get("Accept-Encoding")
		if !strings.Contains(accept, "gzip") {
			logger.WithField("accept-encoding", accept).Error("browser encoding not supported")
			rw.WriteHeader(500)
			return
		}
		rw.Header().Set("Content-Encoding", "gzip")
		h.ServeHTTP(rw, r)
	})
}

var startup = time.Now()

func staticMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		staticLastModified(h)
		h.Set("Cache-Control", "public, max-age=31919000")
		return next(c)
	}
}

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
