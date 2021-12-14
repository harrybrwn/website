package main

import (
	"embed"
	"flag"
	"net"
	"net/http"
	"time"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
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

	if app.Debug {
		godotenv.Load()
	}

	e.HideBanner = true
	db, err := db.Connect(logger)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	jwtConf := &tokenConfig{auth.GenerateECDSATokenConfig()}
	guard := auth.Guard(jwtConf)
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("logger", logger)
			return next(c)
		}
	})
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

	api := e.Group("/api")
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error {
		return c.JSON(200, app.GetQuotes())
	})
	api.GET("/quote", func(c echo.Context) error {
		return c.JSON(200, app.RandomQuote())
	})
	api.POST("/token", token(jwtConf, app.NewUserStore(db)))

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

func token(conf auth.TokenConfig, store app.UserStore) echo.HandlerFunc {
	type userbody struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	return func(c echo.Context) error {
		logger := c.Get("logger").(logrus.FieldLogger)
		ctx := c.Request().Context()
		var body userbody
		err := c.Bind(&body)
		if err != nil {
			logger.WithField("error", err).Error("failed to load user data")
			return err
		}
		logger.WithFields(logrus.Fields{
			"username":  body.Username,
			"email":     body.Email,
			"pw_length": len(body.Password),
		}).Info("getting token")

		if len(body.Password) == 0 {
			logger.Warn("no password")
			return echo.ErrBadRequest
		}
		var u *app.User
		if len(body.Email) > 0 {
			u, err = store.Find(ctx, body.Email)
		} else if len(body.Username) > 0 {
			u, err = store.Find(ctx, body.Username)
		} else {
			return &echo.HTTPError{Code: http.StatusBadRequest, Message: "unable to find username"}
		}
		if err != nil {
			return wrap(err, 404, "could not find user")
		}
		if u == nil {
			logger.Error("returned nil user on store.find")
			return echo.ErrInternalServerError
		}
		err = bcrypt.CompareHashAndPassword(u.PWHash, []byte(body.Password))
		if err != nil {
			return echo.ErrForbidden
		}
		resp, err := auth.NewTokenResponse(conf, &auth.Claims{
			ID:    u.ID,
			UUID:  u.UUID,
			Roles: u.Roles,
		})
		if err != nil {
			return wrap(errors.Wrap(err, "could not generate token"), 500, "Internal server error")
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

func wrap(err error, status int, message string) error {
	return &echo.HTTPError{
		Code:     status,
		Message:  message,
		Internal: err,
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
