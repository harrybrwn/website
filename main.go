//go:build !ci
// +build !ci

package main

import (
	"embed"
	"flag"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sirupsen/logrus"
	"harrybrown.com/app"
	"harrybrown.com/app/chat"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/db"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

//go:generate sh scripts/mockgen.sh

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
	//go:embed build/chatroom/index.html
	chatroomStaticPage []byte
	//go:embed build/invite/index.html
	inviteStaticPage []byte

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
	//go:embed build/sitemap.xml.gz
	sitemapgz []byte

	//go:embed frontend/templates
	templates embed.FS

	logger = log.GetLogger()
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

	e.Logger = log.WrapLogrus(logger)
	e.Debug = app.Debug
	e.DisableHTTP2 = false
	e.HideBanner = true

	if env {
		if err := godotenv.Load(); err != nil {
			logger.WithError(err).Warn("could not load .env")
		}
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

	templates, err := fs.Sub(templates, "frontend")
	if err != nil {
		logger.Fatal(err)
	}

	userStore := app.NewUserStore(db)
	invites := app.NewInvitations(rd, &InvitePathBuilder{"/invite"})

	jwtConf := app.NewTokenConfig()
	guard := auth.Guard(jwtConf)
	withUser := auth.ImplicitUser(jwtConf)
	e.Pre(app.RequestLogRecorder(db, logger))

	e.Any("/", app.Page(harryStaticPage, "index.html"))
	e.GET("/~harry", app.Page(harryStaticPage, "index.html"))
	e.GET("/tanya/hyt", app.Page(hytStaticPage, "harry_y_tanya/index.html"), guard, auth.RoleRequired(auth.RoleTanya))
	e.GET("/remora", app.Page(remoraStaticPage, "remora/index.html"))
	e.GET("/games", app.Page(gamesStaticPage, "games/index.html"), guard)
	e.GET("/admin", app.Page(adminStaticPage, "admin/index.html"), guard, auth.AdminOnly())
	e.GET("/chat/*", app.Page(chatroomStaticPage, "chatroom/index.html"))
	e.GET("/old", echo.WrapHandler(app.HomepageHandler(templates)), guard)

	e.GET("/invite/:id", invitesPageHandler(inviteStaticPage, "text/html", "build/invite/index.html", invites))
	e.POST("/invite/:id", invites.SignUp(userStore))

	e.GET("/static/*", echo.WrapHandler(handleStatic()))
	e.GET("/pub.asc", WrapHandler(keys))
	e.GET("/robots.txt", WrapHandler(robotsHandler))
	e.GET("/sitemap.xml", WrapHandler(sitemapHandler(sitemap, false)))
	e.GET("/sitemap.xml.gz", WrapHandler(sitemapHandler(sitemapgz, true)))
	e.GET("/favicon.ico", faviconHandler())
	e.GET("/manifest.json", json(manifest))

	tokenSrv := app.TokenService{
		Config: jwtConf,
		Tokens: auth.NewRedisTokenStore(auth.RefreshExpiration, rd),
		Users:  userStore,
	}
	emailClient := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	api := e.Group("/api")
	api.POST("/token", tokenSrv.Token)
	api.POST("/refresh", tokenSrv.Refresh)
	api.POST("/revoke", tokenSrv.Revoke, guard)
	api.GET("/info", echo.WrapHandler(web.APIHandler(app.HandleInfo)))
	api.GET("/quotes", func(c echo.Context) error { return c.JSON(200, app.GetQuotes()) })
	api.GET("/quote", func(c echo.Context) error { return c.JSON(200, app.RandomQuote()) })
	api.GET("/bookmarks", json(bookmarks))
	api.GET("/hits", app.Hits(db, app.NewHitsCache(rd), logger))
	api.Any("/ping", WrapHandler(ping))
	api.GET("/runtime", app.HandleRuntimeInfo(app.StartTime), guard, auth.AdminOnly())
	api.GET("/logs", app.LogListHandler(db), guard, auth.AdminOnly())

	chatStore := chat.NewStore(db)
	api.POST("/chat/room", app.CreateChatRoom(chatStore), guard)
	api.GET("/chat/:id/room", app.GetRoom(chatStore), withUser)
	api.GET("/chat/:id/connect", app.ChatRoomConnect(chatStore, rd), withUser)
	api.GET("/chat/:id/messages", app.ListMessages(chatStore), withUser)

	api.POST("/invite/create", invites.Create(), guard)
	api.DELETE("/invite/:id", invites.Delete(), guard)
	api.GET("/invites", invites.List(), guard, auth.AdminOnly())
	api.POST("/mail/send", app.SendMail(emailClient), guard, auth.AdminOnly())

	logger.WithFields(logrus.Fields{"time": app.StartTime}).Info("server starting")
	err = e.Start(net.JoinHostPort("", port))
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
		return c.HTMLBlob(404, notFoundStaticPage)
	}
}

func invitesPageHandler(body []byte, contentType, debugFile string, invitations *app.Invitations) echo.HandlerFunc {
	if app.Debug {
		return func(c echo.Context) error {
			raw, err := os.ReadFile(debugFile)
			if err != nil {
				return err
			}
			ct := http.DetectContentType(raw)
			return invitations.Accept(raw, ct)(c)
		}
	} else {
		return invitations.Accept(body, contentType)
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	staticLastModified(rw.Header())
	rw.Header().Set("Cache-Control", app.StaticCacheControl)
	_, err := rw.Write(gpgPubkey)
	if err != nil {
		logger.WithError(err).Error("could not write response")
	}
}

func faviconHandler() echo.HandlerFunc {
	length := strconv.FormatInt(int64(len(favicon)), 10)
	return func(c echo.Context) error {
		h := c.Response().Header()
		h.Set("Content-Length", length)
		h.Set("Accept-Ranges", "bytes")
		h.Set("Cache-Control", app.StaticCacheControl)
		staticLastModified(h)
		return c.Blob(200, "image/x-icon", favicon)
	}
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

func robotsHandler(rw http.ResponseWriter, r *http.Request) {
	h := rw.Header()
	staticLastModified(h)
	h.Set("Cache-Control", app.StaticCacheControl)
	h.Set("Content-Type", "text/plain")
	_, err := rw.Write(robots)
	if err != nil {
		logger.WithError(err).Error("could not write response body")
	}
}

func sitemapHandler(raw []byte, gzip bool) func(http.ResponseWriter, *http.Request) {
	length := strconv.FormatInt(int64(len(raw)), 10)
	return func(rw http.ResponseWriter, r *http.Request) {
		h := rw.Header()
		staticLastModified(h)
		h.Set("Cache-Control", app.StaticCacheControl)
		h.Set("Content-Length", length)
		h.Set("Content-Type", "text/xml")
		if gzip {
			h.Set("Content-Encoding", "gzip")
		}
		_, err := rw.Write(raw)
		if err != nil {
			logger.WithError(err).Error("could not write response body")
		}
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

func ping(rw http.ResponseWriter, r *http.Request) { rw.WriteHeader(200) }

func staticLastModified(h http.Header) {
	h.Set("Last-Modified", app.StartTime.UTC().Format(http.TimeFormat))
}

func staticCache(h http.Handler) http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		header := rw.Header()
		staticLastModified(header)
		header.Set("Cache-Control", app.StaticCacheControl)
		h.ServeHTTP(rw, r)
	})
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
