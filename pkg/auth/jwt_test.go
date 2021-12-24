package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
	"github.com/pkg/errors"
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
	type table struct {
		errs   []error
		cfg    TokenConfig
		claims Claims
		prep   func(c echo.Context)
	}

	for i, tt := range []table{
		{
			errs: []error{echo.ErrUnauthorized},
			cfg:  GenEdDSATokenConfig(),
			claims: Claims{
				ID:    1,
				UUID:  uuid.New(),
				Roles: []Role{RoleDefault},
				StandardClaims: jwt.StandardClaims{
					Audience: TokenAudience,
					Issuer:   Issuer,
					IssuedAt: time.Now().UTC().Unix(),
					// ExpiresAt: time.Now().UTC().Add(time.Hour).Unix(),
					ExpiresAt: time.Now().UTC().Add(-time.Second).Unix(),
				},
			},
		},
	} {
		if tt.prep == nil {
			tt.prep = func(echo.Context) {}
		}
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			e := echo.New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/protected", nil)
			tok, err := newTokenResp(tt.cfg, &tt.claims)
			is.NoErr(err)
			req.Header.Set("Authorization", fmt.Sprintf("%s %s", tok.TokenType, tok.Token))

			c := e.NewContext(req, rec)
			tt.prep(c)
			fn := func(c echo.Context) error {
				claims := GetClaims(c)
				if claims == nil {
					er := errors.New("no claims in context")
					t.Error(er)
					return er
				}
				is.Equal(claims.ID, tt.claims.ID)
				is.Equal(claims.UUID, tt.claims.UUID)
				is.Equal(claims.Roles, tt.claims.Roles)
				return nil
			}
			err = Guard(tt.cfg)(fn)(c)
			if len(tt.errs) == 0 {
				is.NoErr(err)
			} else {
				for _, er := range tt.errs {
					if !errors.Is(err, er) {
						t.Errorf("expected \"%v\", got \"%v\"", er, err)
					}
				}
			}
		})
	}
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
