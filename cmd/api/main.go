//go:build !ci
// +build !ci

package main

import (
	_ "embed"
	"html/template"
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
	hydra "github.com/ory/hydra-client-go"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"harrybrown.com/app"
	"harrybrown.com/files"
	frontend "harrybrown.com/frontend/legacy"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/email"
	"harrybrown.com/pkg/invite"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	//go :embed build/harrybrwn.com/harry_y_tanya/index.html
	//hytStaticPage []byte
	//go :embed build/harrybrwn.com/admin/index.html
	//adminStaticPage []byte
	//-go:embed build/harrybrwn.com/games/index.html
	//gamesStaticPage []byte
	//-go:embed build/harrybrwn.com/chatroom/index.html
	//chatroomStaticPage []byte
	//go :embed build/harrybrwn.com/invite/index.html
	//inviteStaticPage []byte

	//go :embed files/bookmarks.json
	//bookmarks []byte
	//go :embed build/harrybrwn.com/pub.asc
	//gpgPubkey []byte

	logger = log.SetLogger(log.New(log.WithEnv(), log.WithServiceName("api")))
)

func main() {
	var (
		port = "8080"
		// cookieDomain = getenv("API_TOKEN_COOKIE_DOMAIN", "hrry.local")
		cookieDomain = getenv("API_TOKEN_COOKIE_DOMAIN", "localhost:3000")
		env          []string
		e            = echo.New()
	)
	flag.StringVarP(&port, "port", "p", port, "the port to run the server on")
	flag.StringArrayVar(&env, "env", env, "environment files")
	flag.StringVar(&cookieDomain, "token-cookie-domain", cookieDomain, "domain for cookies")
	flag.BoolVarP(&app.Debug, "debug", "d", app.Debug, "run the app in debug mode")
	flag.Parse()

	logger.SetFormatter(&logrus.JSONFormatter{TimestampFormat: time.RFC3339})
	logger.SetOutput(log.GetOutput("LOG_OUTPUT"))
	e.Logger = log.WrapLogrus(logger)
	e.Debug = app.Debug
	e.DisableHTTP2 = false
	e.HideBanner = true

	if err := godotenv.Load(env...); err != nil {
		logger.WithError(err).Warn("could not load .env")
	}

	if app.Debug {
		// auth.Expiration = time.Second * 30
		logger.SetLevel(logrus.DebugLevel)
	}

	echo.NotFoundHandler = NotFoundHandler()

	db, rd, err := db.Datastores(logger)
	if err != nil {
		logger.Fatal(err)
	}
	defer db.Close()
	defer rd.Close()

	userStore := app.NewUserStore(db)
	var (
		mailer      invite.Mailer
		emailClient *sendgrid.Client
	)
	if emailClient := app.SendgridClient(); emailClient != nil {
		mailer = newInviteMailer(emailClient)
		logger.Info("found sendgrid api key")
	} else {
		logger.Info("emailing disabled: no sendgrid api key")
	}
	invites := app.NewInvitations(rd, &InvitePathBuilder{"/invite"}, mailer)

	jwtConf := app.NewTokenConfig()
	guard := auth.GuardMiddleware(jwtConf)
	e.Pre(echo.WrapMiddleware(web.AccessLog(logger)))
	e.Use(echo.WrapMiddleware(web.Metrics()))

	e.GET("/metrics", WrapHandler(web.MetricsHandler().ServeHTTP))

	// e.GET("/tanya/hyt", app.Page(hytStaticPage, "harrybrwn.com/harry_y_tanya/index.html"), guard, auth.RoleRequired(auth.RoleTanya))
	// e.GET("/admin", app.Page(adminStaticPage, "harrybrwn.com/admin/index.html"), guard, auth.AdminOnly())
	// e.GET("/chat/*", app.Page(chatroomStaticPage, "chatroom/index.html"))

	// e.GET("/invite/:id", invitesPageHandler(inviteStaticPage, "text/html", "build/invite/index.html", invites))
	// e.POST("/invite/:id", invites.SignUp(userStore))

	tokenSrv := app.TokenService{
		Config:       jwtConf,
		Tokens:       auth.NewRedisTokenStore(auth.RefreshExpiration, rd),
		Users:        userStore,
		HydraAdmin:   hydra.NewAPIClient(app.HydraAdminConfig()).AdminApi,
		CookieDomain: cookieDomain,
	}
	api := e.Group("/api")
	api.POST("/token", tokenSrv.Token)
	api.POST("/refresh", tokenSrv.Refresh)
	api.POST("/revoke", tokenSrv.Revoke, guard)
	api.POST("/login", tokenSrv.Login, auth.ImplicitUser(jwtConf))
	api.POST("/consent", app.ConsentHandler(tokenSrv.HydraAdmin, userStore), guard)
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error { return c.JSON(200, app.GetQuotes()) })
	api.GET("/quote", func(c echo.Context) error { return c.JSON(200, app.RandomQuote()) })
	api.GET("/bookmarks", json(files.Bookmarks))
	api.GET("/hits", app.Hits(db, app.NewHitsCache(rd), logger))
	api.Any("/ping", WrapHandler(ping))
	api.GET("/runtime", app.HandleRuntimeInfo(app.StartTime), guard, auth.AdminOnly())
	api.GET("/logs", app.LogListHandler(db), guard, auth.AdminOnly())
	api.Any("/health/ready", app.Ready(db, rd))
	api.Any("/health/alive", app.Alive)
	api.OPTIONS("/token", WrapHandler(HandleCORS))
	api.OPTIONS("/consent", WrapHandler(HandleCORS))
	api.OPTIONS("/login", WrapHandler(HandleCORS))
	api.Any("/demo", func(ctx echo.Context) error {
		HandleCORS(ctx.Response(), ctx.Request())
		d := ctx.QueryParam("domain")
		if len(d) == 0 {
			d = "hrry.local"
		}
		var ss http.SameSite
		switch ctx.QueryParam("ss") {
		case "none":
			ss = http.SameSiteNoneMode
		case "lax":
			ss = http.SameSiteLaxMode
		case "strict":
			ss = http.SameSiteStrictMode
		default:
			ss = http.SameSiteNoneMode
		}
		secure, err := strconv.ParseBool(ctx.QueryParam("secure"))
		if err != nil {
			secure = true
		}
		httpOnly, err := strconv.ParseBool(ctx.QueryParam("httponly"))
		if err != nil {
			httpOnly = false
		}
		path := ctx.QueryParam("path")
		if len(path) == 0 {
			path = "/"
		}

		http.SetCookie(ctx.Response(), &http.Cookie{
			Name:  "demo",
			Value: time.Now().String(),
			// Expires:  time.Now().Add(time.Minute * 5),
			Path:     path,
			Domain:   d,
			Secure:   secure,
			SameSite: ss,
			HttpOnly: httpOnly,
		})
		return ctx.JSON(200, map[string]any{
			"status":  200,
			"message": "this is a demo",
		})
	})

	//withUser := auth.ImplicitUser(jwtConf)
	//chatStore := chat.NewStore(db)
	//api.POST("/chat/room", app.CreateChatRoom(chatStore), guard)
	//api.GET("/chat/:id/room", app.GetRoom(chatStore), withUser)
	//api.GET("/chat/:id/connect", app.ChatRoomConnect(chatStore, rd), withUser)
	//api.GET("/chat/:id/messages", app.ListMessages(chatStore), withUser)

	api.POST("/invite/create", invites.Create(), guard)
	api.DELETE("/invite/:id", invites.Delete(), guard)
	api.GET("/invites", invites.List(), guard, auth.AdminOnly())
	api.POST("/mail/send", app.SendMail(emailClient), guard, auth.AdminOnly())

	logger.WithFields(logrus.Fields{"time": app.StartTime}).Info("server starting")
	if web.SSLCertificateFileFlag != "" && web.SSLKeyFileFlag != "" {
		err = e.StartTLS(
			net.JoinHostPort("", port),
			web.SSLCertificateFileFlag,
			web.SSLKeyFileFlag,
		)
	} else {
		err = e.Start(net.JoinHostPort("", port))
	}
	if err != nil {
		logger.Fatal(err)
	}
}

func NotFoundHandler() echo.HandlerFunc {
	if app.Debug {
		return func(c echo.Context) error {
			if strings.HasPrefix(c.Request().RequestURI, "/api") {
				return echo.ErrNotFound
			}
			return app.ServeFile(c, 404, "build/pages/404.html")
		}
	}
	return func(c echo.Context) error {
		if strings.HasPrefix(c.Request().RequestURI, "/api") {
			return echo.ErrNotFound
		}
		return c.HTMLBlob(404, frontend.NotFoundHTML)
	}
}

func HandleCORS(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Add(
		"Access-Control-Allow-Headers",
		"Content-Type, Authorization, Cookie, Origin",
	)
}

func newInviteMailer(client *sendgrid.Client) invite.Mailer {
	m, err := invite.NewMailer(
		email.Email{Name: "Harry Brown", Address: "admin@harrybrwn.com"},
		"You're Invited!",
		template.Must(template.New("email-invite").Parse(string(frontend.InviteEmailHTML))),
		client,
	)
	if err != nil {
		logger.Fatal(err)
	}
	return m
}

func json(raw []byte) echo.HandlerFunc {
	return func(c echo.Context) error {
		h := c.Response().Header()
		staticLastModified(h)
		h.Set("Cache-Control", app.StaticCacheControl)
		h.Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
		return c.Blob(200, "application/json", raw)
	}
}

func ping(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(200) }

func staticLastModified(h http.Header) {
	h.Set("Last-Modified", app.StartTime.UTC().Format(http.TimeFormat))
}

func WrapHandler(h http.HandlerFunc) echo.HandlerFunc {
	return echo.WrapHandler(h)
}

type InvitePathBuilder struct{ p string }

func (ipb *InvitePathBuilder) Path(id string) string {
	return filepath.Join("/", ipb.p, id)
}

func (ipb *InvitePathBuilder) GetID(r *http.Request) string {
	list := strings.Split(r.URL.Path, string(filepath.Separator))
	return list[2]
}

func mustSub(sys fs.FS, dir string) fs.FS {
	f, err := fs.Sub(sys, dir)
	if err != nil {
		logger.Fatal(err)
	}
	return f
}

func getenv(key, defaultValue string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		logger.WithField("key", key).Infof("using ENV value %s", defaultValue)
		return defaultValue
	}
	logger.WithField("key", key).Infof("using ENV value %s", v)
	return v
}
