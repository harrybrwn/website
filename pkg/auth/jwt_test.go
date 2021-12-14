package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
)

func Test(t *testing.T) {
	is := is.New(t)
	// key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	conf := GenerateECDSATokenConfig()
	now := time.Now().UTC()
	tok := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.StandardClaims{
		ExpiresAt: now.Add(time.Hour).Unix(),
		IssuedAt:  now.Unix(),
	})
	token, err := tok.SignedString(conf.Private())
	is.NoErr(err)
	// claims := jwt.StandardClaims{}
	claims := Claims{}
	parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
		return conf.Public(), nil
		// return &conf.(*tokenConfig).key.PublicKey, nil
	})
	is.NoErr(err)
	is.True(parsed.Valid)
	is.Equal(now.Add(time.Hour).Unix(), claims.ExpiresAt)
	is.Equal(now.Unix(), claims.IssuedAt)
	time.Sleep(time.Millisecond * 10)
	fmt.Println(now.Unix(), time.Now().UTC().Unix())
	fmt.Println(now.Unix() < time.Now().UTC().Add(Expiration).Unix())
}

func TestLogin(t *testing.T) {
	e := echo.New()
	// mid := middleware.JWT("key")
	// e.Use(mid)
	req := httptest.NewRequest("POST", "/login", asBody(map[string]string{
		"username": "testuser",
		"password": "pw",
	}))
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetPath("/login")
	h := func(c echo.Context) error {
		// fmt.Println(c.Request().RequestURI)
		return nil
	}
	err := h(c)
	if err != nil {
		t.Fatal(err)
	}
}

func asBody(v interface{}) io.Reader {
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(v)
	return &b
}
