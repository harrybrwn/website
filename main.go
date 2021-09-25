package main

import (
	"embed"
	_ "embed"
	"fmt"
	"net/http"

	"harrybrown.com/app"
	"harrybrown.com/pkg/cmd"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	router = web.NewRouter()
	port   = "8080"
)

var (
	//go:embed embeds/harry.html
	harryStaticPage []byte

	//go:embed embeds/keys/pub.asc
	pubkey []byte

	//go:embed static/css static/data static/files static/img static/js
	static embed.FS
)

func init() {
	app.StringFlag(&port, "port", "the port to run the server on")
	app.ParseFlags()

	router.HandleRoutes(app.Routes)
}

func main() {
	if app.Debug {
		log.Printf("running on localhost:%s\n", port)

		app.Commands = append(app.Commands, cmd.Command{
			Syntax:      "addr",
			Description: "show server address",
			Run:         func() { fmt.Printf("localhost:%s\n", port) },
		})

		// scans stdin and runs the commands given alongside the server
		go cmd.Run(app.Commands)
	}

	if app.Debug {
		router.AddRoute("/static/", app.NewFileServer("static")) // handle file server
	} else {
		router.AddRoute("/static/", http.FileServer(http.FS(static)))
	}

	router.HandleFunc("/pub.asc", keys)

	router.AddRoute("/~harry", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(harryStaticPage)
	}))

	if err := router.ListenAndServe(":" + port); err != nil {
		log.Fatal(err)
	}
}

func keys(rw http.ResponseWriter, r *http.Request) {
	rw.Write(pubkey)
}
