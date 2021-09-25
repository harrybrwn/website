package main

import (
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

	router.HandleFunc("/keys", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write([]byte("not done with this feature yet..."))
	})

	router.AddRoute("/~harry", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(harryStaticPage)
	}))

	if err := router.ListenAndServe(":" + port); err != nil {
		log.Fatal(err)
	}
}
