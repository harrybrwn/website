package app

import (
	"html/template"
	"io/fs"
	"net/http"
	"strconv"

	"harrybrown.com/pkg/log"
)

// Debug cooresponds with the debug flag
var Debug = false

func init() {
	BoolFlag(&Debug, "debug", "turn on debugging options")
}

func HomepageHandler(fs fs.FS) http.Handler {
	type Data struct {
		Title string
		Age   string
		Quote Quote
	}
	t, err := template.ParseFS(fs, "*/pages/home.html", "*/index.html", "*/nav.html")
	if err != nil {
		panic(err) // panic on server startup
	}
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		err := t.ExecuteTemplate(rw, "base", &struct {
			Title string
			Data  Data
		}{
			Data: Data{
				Age:   strconv.FormatInt(int64(GetAge()), 10),
				Quote: RandomQuote(),
			},
			Title: "Harry Brown",
		})
		if err != nil {
			log.Error(err)
			_, err = rw.Write([]byte("something went wrong"))
			if err != nil {
				logger.WithError(err).Error("failed to write error message to response")
			}
			rw.WriteHeader(500)
		}
	})
}
