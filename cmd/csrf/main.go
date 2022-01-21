package main

import (
	_ "embed"
	"net/http"
)

//go:embed index.html
var index []byte

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		rw.Write(index)
	})
	http.ListenAndServe(":8090", nil)
}
