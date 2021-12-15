package main

import (
	"embed"
	"flag"
	"net"
	"net/http"
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
	//go:embed embeds/harry.html
	harryStaticPage []byte
	//go:embed public/pub.asc
	pubkey []byte
	//go:embed public/robots.txt
	robots []byte
	//go:embed public/favicon.ico
	favicon []byte

	//go:embed static/css static/data static/files static/img static/js
	static embed.FS

	//go:embed templates
	templates embed.FS

	logger = logrus.New()
)

func main() {
	var (
		port = "8080"
		e    = echo.New()
	)
	e.Logger = log.WrapLogrus(logger)
	flag.StringVar(&port, "port", port, "the port to run the server on")
	flag.Parse()

	if app.Debug {
		godotenv.Load()
	}

	e.HideBanner = true
	db, err := db.Connect(logger)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	jwtConf := &tokenConfig{auth.GenEdDSATokenConfig()}
	guard := auth.Guard(jwtConf)
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
	}, guard)
	e.GET("/secret/admin", func(c echo.Context) error {
		return c.HTML(200, `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta http-equiv="X-UA-Compatible" content="IE=edge">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Secret</title>
</head>
<body>
	<h1>This is a Secret for only the admins</h1>
</body>
</html>`)
	}, guard, admin())

	e.GET("/old", echo.WrapHandler(app.HomepageHandler(templates)), guard)

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

type tokenConfig struct {
	auth.TokenConfig
}

func (tc *tokenConfig) GetToken(r *http.Request) (string, error) {
	c, err := r.Cookie("token")
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
		ctx := c.Request().Context()
		var (
			err  error
			body userbody
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
		c.SetCookie(&http.Cookie{
			Name:    "token",
			Value:   resp.Token,
			Expires: time.Unix(resp.Expires, 0),
			Path:    "/",
		})
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
	if app.Debug {
		return http.StripPrefix("/static/", http.FileServer(http.Dir("static")))
	}
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
