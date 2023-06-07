package main

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	flag "github.com/spf13/pflag"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	logger = log.SetLogger(log.New(
		log.WithEnv(),
		log.WithFormat(log.JSONFormat),
		log.WithServiceName("vanity-imports"),
	))
	domain = "gopkg.hrry.dev"
	home   = "https://hrry.me"
	repo   = "https://github.com/harrybrwn"
)

func main() {
	port := 8085
	flag.IntVarP(&port, "port", "p", port, "port to run the server on")
	flag.StringVar(&domain, "domain", domain, "domain for packages")
	flag.StringVar(&repo, "repo", repo, "root package repo")
	flag.Parse()

	v := Vanity{
		Domain:  domain,
		RepoURL: repo,
	}
	r := chi.NewRouter()
	r.Use(web.AccessLog(logger))
	r.Use(web.Metrics())
	r.Get("/*", VanityImport(&v))
	r.Handle("/metrics", web.MetricsHandler())
	r.Head("/health/ready", func(w http.ResponseWriter, r *http.Request) {})
	addr := fmt.Sprintf(":%d", port)
	if err := web.ListenAndServe(addr, r); err != nil {
		logger.WithError(err).Fatal("listen and serve failed")
	}
}

type Vanity struct {
	RepoURL string
	Domain  string
	Package *Package
}

type Package struct {
	Name string
	VCS  string
}

func VanityImport(vanity *Vanity) func(http.ResponseWriter, *http.Request) {
	t, err := template.New("base").Parse(importPage)
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("go-get") != "1" {
			w.Header().Set("Location", home)
			w.WriteHeader(http.StatusFound)
			return
		}
		var (
			err error
			v   = *vanity
		)
		v.Package = &Package{VCS: "git"}
		v.Package.Name, err = packageName(r)
		if err != nil {
			w.WriteHeader(404)
			return
		}
		logger.WithField("headers", r.Header).Info("got package request")
		err = t.Execute(w, &v)
		if err != nil {
			logger.WithError(err).Error("failed to execute template")
			return
		}
	}
}

var (
	errNoPackage   = errors.New("no package name")
	errInvalidPath = errors.New("invalid url path")
)

const importPage = `<!DOCTYPE html>
<html lang="en">
<head>
	<link rel="icon" href="data:;base64,iVBORw0KGgo=">
	<meta name="go-import" content="{{.Domain}}/{{.Package.Name}} {{.Package.VCS}} {{.RepoURL}}/{{.Package.Name}}">
</head>
<body>
	go get {{.Domain}}/{{.Package.Name}}
</body>
</html>`

func packageName(r *http.Request) (string, error) {
	p := r.URL.Path
	if len(p) == 0 || p[0] != '/' {
		return "", errInvalidPath
	}
	parts := strings.Split(p, "/")
	if len(parts) < 2 || len(parts[1]) == 0 {
		return "", errNoPackage
	}
	return strings.Join(parts[1:], "/"), nil
}
