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
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
)

func TestTokenConfig(t *testing.T) {
	is := is.New(t)
	for _, conf := range []TokenConfig{
		GenEdDSATokenConfig(),
		GenerateECDSATokenConfig(),
	} {
		now := time.Now().UTC()
		tok := jwt.NewWithClaims(conf.Type(), jwt.StandardClaims{
			ExpiresAt: now.Add(time.Hour).Unix(),
			IssuedAt:  now.Unix(),
		})
		token, err := tok.SignedString(conf.Private())
		is.NoErr(err)
		claims := Claims{}
		parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			return conf.Public(), nil
		})
		is.NoErr(err)
		is.True(parsed.Valid)
		is.Equal(now.Add(time.Hour).Unix(), claims.ExpiresAt)
		is.Equal(now.Unix(), claims.IssuedAt)
	}
}

func TestGuard(t *testing.T) {
	is := is.New(t)
	e := echo.New()
	conf := GenEdDSATokenConfig()
	e.Use(Guard(conf))
	e.GET("/protected", func(c echo.Context) error {
		claims := GetClaims(c)
		is.True(claims != nil)
		return nil
	})
	tok := newToken(t, conf)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/protected", nil)
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", tok.TokenType, tok.Token))
	c := e.NewContext(req, rec)
	e.Router().Find("GET", "/protected", c)
}

func TestLogin(t *testing.T) {
	e := echo.New()
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

func newToken(t *testing.T, conf TokenConfig) *TokenResponse {
	t.Helper()
	resp, err := NewTokenResponse(conf, &Claims{
		ID:    1,
		UUID:  uuid.New(),
		Roles: []Role{RoleDefault},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func testingToken(conf TokenConfig) func(echo.Context) error {
	return func(c echo.Context) error {
		resp, err := NewTokenResponse(conf, &Claims{
			ID:    1,
			UUID:  uuid.New(),
			Roles: []Role{RoleDefault},
		})
		if err != nil {
			return err
		}
		return c.JSON(200, resp)
	}
}

func asBody(v interface{}) io.Reader {
	var b bytes.Buffer
	json.NewEncoder(&b).Encode(v)
	return &b
}
