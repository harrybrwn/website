package main

import (
	"embed"
	"flag"
	"io/fs"
	"net"
	"net/http"
	"strconv"
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

//go:generate go run ./cmd/key-gen -o ./embeds/jwt-signer
//go:generate yarn build

var (
	//go:embed embeds/jwt-signer.pub
	pubkeyPem []byte
	//go:embed embeds/jwt-signer.key
	privkeyPem []byte

	//go:embed build/index.html
	harryStaticPage []byte
	//go:embed build/pub.asc
	gpgPubkey []byte
	//go:embed build/robots.txt
	robots []byte
	//go:embed build/favicon.ico
	favicon []byte
	//go:embed build/static
	static embed.FS

	//go:embed templates
	templates embed.FS

	logger = logrus.New()
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
	e.GET("/", echo.WrapHandler(harry()))
	e.GET("/static/*", echo.WrapHandler(handleStatic()))
	e.GET("/pub.asc", echo.WrapHandler(http.HandlerFunc(keys)))
	e.GET("/~harry", echo.WrapHandler(harry()))
	e.GET("/robots.txt", echo.WrapHandler(http.HandlerFunc(robotsHandler)))
	e.GET("/favicon.ico", func(c echo.Context) error {
		c.Response().Header().Set("Cache-Control", "public, max-age=31919000")
		return c.Blob(200, "image/x-icon", favicon)
	})
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
	}, guard, admin())
	e.GET("/balls", func(c echo.Context) error {
		return c.HTML(200, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Secret</title>
</head>
<body>
	<canvas id="canvas"></canvas>
	<script src="/static/js/balls.js"></script>
</body>
</html>`)
	})

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error {
		return c.JSON(200, app.GetQuotes())
	})
	api.GET("/quote", func(c echo.Context) error {
		return c.JSON(200, app.RandomQuote())
	})
	api.POST("/token", TokenHandler(jwtConf, app.NewUserStore(db)))

	logger.WithField("time", startup).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
	if err != nil {
		log.Fatal(err)
	}
}

const tokenKey = "_token"

func admin() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := auth.GetClaims(c)
			if claims == nil {
				return echo.ErrForbidden
			}
			for _, r := range claims.Roles {
				if r == auth.RoleAdmin {
					return next(c)
				}
			}
			return echo.ErrForbidden
		}
	}
}

func NewTokenConfig() auth.TokenConfig {
	conf, err := auth.NewEdDSATokenConfig(privkeyPem, pubkeyPem)
	if err != nil {
		panic(err) // happens at startup
	}
	return &tokenConfig{conf}
}

type tokenConfig struct {
	auth.TokenConfig
}

func (tc *tokenConfig) GetToken(r *http.Request) (string, error) {
	c, err := r.Cookie(tokenKey)
	if err != nil {
		return auth.GetBearerToken(r)
	}
	return c.Value, nil
}

func TokenHandler(conf auth.TokenConfig, store app.UserStore) echo.HandlerFunc {
	type userbody struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(c echo.Context) error {
		var (
			err         error
			body        userbody
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
		req.URL.Query().Get("cookie")
		logger.WithFields(logrus.Fields{
			"username": body.Username,
			"email":    body.Email,
		}).Info("getting token")
		if len(body.Password) == 0 {
			return failure(http.StatusBadRequest, "user gave zero length password")
		}

		var u *app.User
		if len(body.Email) > 0 {
			u, err = store.Find(ctx, body.Email)
		} else if len(body.Username) > 0 {
			u, err = store.Find(ctx, body.Username)
		} else {
			return failure(404, "unable to find user")
		}
		if err != nil {
			return wrap(err, 404, "could not find user")
		}
		if u == nil {
			return failure(404, "user store returned nil user on store.Find")
		}
		err = u.VerifyPassword(body.Password)
		if err != nil {
			return wrap(err, http.StatusForbidden)
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
	rw.Header().Set("Cache-Control", "public, max-age=31919000")
	rw.Write(gpgPubkey)
}

func harry() http.Handler {
	if app.Debug {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			rw.Header().Set("Content-Type", "text/html")
			rw.Header().Set("Cache-Control", "public, max-age=31919000")
			http.ServeFile(rw, r, "build/index.html")
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
	if app.Debug {
		return http.StripPrefix("/static/", http.FileServer(http.Dir("build/static")))
	}
	fs, err := fs.Sub(static, "build")
	if err != nil {
		fs = static
	}
	return staticCache(http.FileServer(http.FS(fs)))
}

var startup = time.Now()

func staticCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Last-Modified", startup.Format(time.RFC1123))
		rw.Header().Set("Cache-Control", "public, max-age=31919000")
		h.ServeHTTP(rw, r)
	})
}
