package main

import (
	"errors"
	"flag"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
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
	repo   = "https://github.com/harrybrwn"
)

func main() {
	flag.StringVar(&domain, "domain", domain, "domain for packages")
	flag.StringVar(&repo, "repo", repo, "root package repo")
	flag.Parse()

	v := Vanity{
		Domain:  domain,
		RepoURL: repo,
	}
	r := chi.NewRouter()
	r.Use(web.AccessLog(logger))
	r.Get("/*", VanityImport(&v))
	addr := ":8085"
	logger.WithField("address", addr).Info("starting server")
	http.ListenAndServe(addr, r)
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
			w.Header().Set("Location", "https://hrry.me")
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
		err = t.Execute(w, &v)
		if err != nil {
			logger.WithError(err).Error("failed to execute template")
			return
		}
	}
}

const importPage = `<!DOCTYPE html>
<html lang="en">
<head>
<<<<<<< HEAD
	<link rel="icon" href="data:;base64,iVBORw0KGgo=">
=======
>>>>>>> 347d9a6ec6f7a812be28eb774551b67638605813
	<meta name="go-import" content="{{.Domain}}/{{.Package.Name}} {{.Package.VCS}} {{.RepoURL}}/{{.Package.Name}}">
</head>
<body>
	go get {{.Domain}}/{{.Package.Name}}
</body>
</html>`

const userPage = `<!DOCTYPE html>
<html lang="en">
<head>
</head>
<body>
</body>
</html>`

func packageName(r *http.Request) (string, error) {
	p := r.URL.Path
	if p[0] == '/' {
		p = p[1:]
	}
	parts := strings.Split(p, string(filepath.Separator))
	if len(parts) < 1 {
		return "", errors.New("no package name")
	}
	return parts[0], nil
}
