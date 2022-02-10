package main

import (
	_ "embed"
	"log"
	"net/http"
)

//go:embed index.html
var index []byte

func main() {
	http.HandleFunc("/", func(rw http.ResponseWriter, r *http.Request) {
		_, err := rw.Write(index)
		if err != nil {
			log.Println(err)
		}
	})
	log.Fatal(http.ListenAndServe(":8090", nil))
}
