package main

import (
	"fmt"

	"harrybrown.com/app"
	"harrybrown.com/pkg/auth"
	"harrybrown.com/pkg/cmd"
	"harrybrown.com/pkg/log"
	"harrybrown.com/pkg/web"
)

var (
	router = web.NewRouter()
	port   = "8080"
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

	if err := router.ListenAndServe(":" + port); err != nil {
		log.Fatal(err)
	}
}
