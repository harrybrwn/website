package auth

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

// RedirectHandler handles Oauth 2.0 redirects.
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("error") != "" {
		jsonerr := map[string]string{
			"error":       query.Get("error"),
			"description": query.Get("error_description"),
		}
		w.WriteHeader(404)
		json.NewEncoder(w).Encode(jsonerr)
		return
	}

	var jsonResp = map[string]string{"error": "redirect server refused request"}
	if validate(r) {
		log.Println("redirect login successful")
		jsonResp = map[string]string{"code": query.Get("code")}
		w.WriteHeader(200)
	} else {
		log.Println("redirect login failed:", query, os.Getenv("HARRYBRWN_REDIRECTS_KEY"))
		w.WriteHeader(403)
	}
	json.NewEncoder(w).Encode(jsonResp)
}

func validate(r *http.Request) bool {
	if key, ok := r.URL.Query()["state"]; ok {
		hash, err := base64.StdEncoding.DecodeString(key[0])
		if err != nil {
			return false
		}
		err = bcrypt.CompareHashAndPassword(
			hash,
			[]byte(os.Getenv("HARRYBRWN_REDIRECTS_KEY")),
		)
		log.Println(err)
		if err == nil {
			return true
		}
	}
	return false
}
