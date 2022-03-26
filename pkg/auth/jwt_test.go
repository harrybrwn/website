package auth

import (
	"bytes"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
	"github.com/pkg/errors"
)

func TestTokenConfigs(t *testing.T) {
	is := is.New(t)
	for _, conf := range []TokenConfig{
		GenEdDSATokenConfig(),
		GenerateECDSATokenConfig(),
	} {
		now := time.Now().UTC()
		tok := jwt.NewWithClaims(conf.Type(), jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		})
		token, err := tok.SignedString(conf.Private())
		is.NoErr(err)
		claims := Claims{}
		parsed, err := jwt.ParseWithClaims(token, &claims, func(t *jwt.Token) (interface{}, error) {
			return conf.Public(), nil
		})
		is.NoErr(err)
		is.True(parsed.Valid)
		is.Equal(jwt.NewNumericDate(now.Add(time.Hour)), jwt.NewNumericDate(claims.ExpiresAt.UTC()))
		is.Equal(jwt.NewNumericDate(now), jwt.NewNumericDate(claims.IssuedAt.UTC()))
	}
}

func TestGuard(t *testing.T) {
	type table struct {
		errs   []error
		cfg    TokenConfig
		claims Claims
	}

	for i, tt := range []table{
		{
			// Negative expires_at
			errs: []error{echo.ErrUnauthorized},
			cfg:  GenEdDSATokenConfig(),
			claims: Claims{
				ID:    1,
				UUID:  uuid.New(),
				Roles: []Role{RoleDefault},
				RegisteredClaims: jwt.RegisteredClaims{
					Audience:  []string{TokenAudience},
					Issuer:    Issuer,
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
					ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(-10 * time.Second)),
				},
			},
		},
		{
			errs: []error{ErrNoAudience},
			cfg:  GenerateECDSATokenConfig(),
			claims: Claims{
				ID: 90,
				RegisteredClaims: jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
					Issuer:    Issuer,
					Audience:  []string{}, // invalid audience
				},
			},
		},
		{
			errs: []error{ErrBadIssuerOrAud, echo.ErrBadRequest},
			cfg:  GenEdDSATokenConfig(),
			claims: Claims{
				ID: 12,
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    "wrong issuer",
					Audience:  []string{TokenAudience},
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
					ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
				},
			},
		},
		{
			errs: []error{ErrBadIssuerOrAud, echo.ErrBadRequest},
			cfg:  GenEdDSATokenConfig(),
			claims: Claims{
				ID: 12,
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    Issuer,
					Audience:  []string{"wrong audience"},
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
					ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
				},
			},
		},
		{
			errs: []error{},
			cfg:  GenEdDSATokenConfig(),
			claims: Claims{
				ID:    3,
				UUID:  uuid.New(),
				Roles: []Role{RoleDefault},
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    Issuer,
					Audience:  []string{TokenAudience},
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
					ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
				},
			},
		},
		{
			errs: []error{errFailingTokenConfig, echo.ErrUnauthorized},
			cfg:  &failingTokenConfig{GenEdDSATokenConfig()},
			claims: Claims{
				ID:    3,
				UUID:  uuid.New(),
				Roles: []Role{RoleDefault},
				RegisteredClaims: jwt.RegisteredClaims{
					Issuer:    Issuer,
					Audience:  []string{TokenAudience},
					IssuedAt:  jwt.NewNumericDate(time.Now().UTC()),
					ExpiresAt: jwt.NewNumericDate(time.Now().UTC().Add(time.Minute)),
				},
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			e := echo.New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/protected", nil)
			tok, err := newTokenResp(tt.cfg, &tt.claims)
			is.NoErr(err)
			req.Header.Set("Authorization", fmt.Sprintf("%s %s", tok.TokenType, tok.Token))

			c := e.NewContext(req, rec)
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
			is.True(GetClaims(c) == nil)
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
			resp, err := NewTokenResponse(tt.cfg, &tt.claims)
			is.NoErr(err)
			is.True(resp.Expires.After(time.Now()))
			is.True(resp.RefreshToken != "")
		})
	}
}

func TestValidateRefreshToken(t *testing.T) {
	type table struct {
		errs    []error
		vErr    uint32
		refresh string
	}
	cfg := GenEdDSATokenConfig()
	keyfunc := func(*jwt.Token) (interface{}, error) { return cfg.Public(), nil }
	genToken := func(ex time.Time, aud, iss string) string {
		return generateRefreshToken(cfg, ex, []Role{RoleDefault}, aud, iss)
	}
	for i, tt := range []table{
		{
			refresh: genToken(time.Now().Add(time.Hour*5), refreshAudience, Issuer),
		},
		{
			refresh: genToken(time.Date(2020, time.January, 5, 4, 3, 2, 1, time.Local), refreshAudience, Issuer),
			vErr:    jwt.ValidationErrorExpired,
		},
		{
			refresh: genToken(time.Now().Add(time.Hour), "_not_a_refresh_aud", Issuer),
			errs:    []error{ErrBadRefreshTokenAud},
		},
		{
			refresh: genToken(time.Now().Add(time.Hour), refreshAudience, "_"+Issuer),
			errs:    []error{ErrBadIssuer},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			_, err := ValidateRefreshToken(tt.refresh, keyfunc)
			if tt.vErr != 0 {
				validationErr, ok := err.(*jwt.ValidationError)
				if !ok {
					t.Fatal("expected a jwt validation error")
				}
				if validationErr.Errors&tt.vErr == 0 {
					t.Errorf("expecting validation error %d", tt.vErr)
				}
			} else if len(tt.errs) == 0 {
				if err != nil {
					t.Fatal(err)
				}
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

func TestGetBearer_Err(t *testing.T) {
	for _, header := range []http.Header{
		{},
		{"Authorization": {"what?"}},
	} {
		r := http.Request{Header: header}
		_, err := GetBearerToken(&r)
		if err != errAuthHeaderTokenMissing {
			t.Fatalf("expected error %v, got %v", errAuthHeaderTokenMissing, err)
		}
	}
}

func TestAdminOnly(t *testing.T) {
	type table struct {
		claims   *Claims
		err      error
		executed bool
	}

	is := is.New(t)
	for _, tt := range []table{
		{
			err:      echo.ErrForbidden,
			claims:   nil,
			executed: false,
		},
		{
			err:      echo.ErrForbidden,
			claims:   &Claims{Roles: []Role{RoleDefault}},
			executed: false,
		},
		{
			err:      nil,
			claims:   &Claims{Roles: []Role{RoleDefault, RoleAdmin}},
			executed: true,
		},
	} {
		e := echo.New()
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/", nil)
		c := e.NewContext(req, rec)
		c.Set(ClaimsContextKey, tt.claims)
		executed := false
		next := func(c echo.Context) error {
			executed = true
			return nil
		}
		err := AdminOnly()(next)(c)
		if !errors.Is(err, tt.err) {
			t.Errorf("expected \"%v\", got \"%v\"", tt.err, err)
		}
		is.Equal(executed, tt.executed)
	}
}

func TestRole_ScanValue(t *testing.T) {
	var (
		is  = is.New(t)
		r   Role
		v   driver.Value
		err error
	)
	for _, s := range []string{
		"admin",
		"default",
		"family",
		"tanya",
	} {
		is.NoErr(r.Scan(s))
		is.Equal(r, ParseRole(s))
		is.Equal(r.String(), s)
		v, err = r.Value()
		is.NoErr(err)
		is.Equal(v, r)
		is.Equal(v, ParseRole(s))
	}

	for _, s := range []string{
		"",
		"hello",
		"____",
	} {
		r = RoleAdmin                       // r must change for the next checks to pass
		is.Equal(r.Scan(s), ErrInvalidRole) // should fail with invalid role
		is.Equal(r, RoleInvalid)
		v, err = r.Value()
		is.Equal(err, ErrInvalidRole)
		is.Equal(v, RoleInvalid)
	}
	is.NoErr(r.Scan(int8(1)))
	is.NoErr(r.Scan(int16(1)))
	is.NoErr(r.Scan(int32(1)))
	is.NoErr(r.Scan(int64(1)))
	is.NoErr(r.Scan(int(1)))
	is.NoErr(r.Scan(uint8(1)))
	is.NoErr(r.Scan(uint16(1)))
	is.NoErr(r.Scan(uint32(1)))
	is.NoErr(r.Scan(uint64(1)))
	is.NoErr(r.Scan(uint(1)))
	is.NoErr(r.Scan([]byte("admin")))
	is.True(errors.Is(r.Scan(float32(1)), ErrInvalidRole))
	is.True(errors.Is(r.Scan(float64(1)), ErrInvalidRole))
	is.True(errors.Is(r.Scan(complex64(1)), ErrInvalidRole))
	is.True(errors.Is(r.Scan(complex128(1)), ErrInvalidRole))

	r = Role(65000)
	is.Equal(r.String(), "")
}

func TestRoleRequired(t *testing.T) {
	e := echo.New()
	for i, tst := range []func(t *testing.T, c echo.Context){
		func(t *testing.T, c echo.Context) {
			h := RoleRequired(RoleFamily)
			err := h(func(echo.Context) error {
				t.Fatal("should not execute")
				return nil
			})(c)
			if !errors.Is(echo.ErrForbidden, err) {
				t.Error("wrong error")
			}
		},
		func(t *testing.T, c echo.Context) {
			h := RoleRequired(RoleFamily)
			c.Set(ClaimsContextKey, &Claims{Roles: []Role{RoleDefault}})
			err := h(func(echo.Context) error {
				t.Fatal("should not execute")
				return nil
			})(c)
			if !errors.Is(echo.ErrForbidden, err) {
				t.Fatal("wrong error")
			}
		},
		func(t *testing.T, c echo.Context) {
			h := RoleRequired(RoleAdmin)
			c.Set(ClaimsContextKey, &Claims{Roles: []Role{RoleAdmin}})
			err := h(func(echo.Context) error {
				return ErrInvalidRole
			})(c)
			if !errors.Is(ErrInvalidRole, err) {
				t.Fatal("wrong error")
			}
		},
		func(t *testing.T, c echo.Context) {
			h := RoleRequired(RoleAdmin)
			c.Set(ClaimsContextKey, &Claims{Roles: []Role{RoleAdmin}})
			err := h(func(echo.Context) error { return nil })(c)
			if err != nil {
				t.Error("should get nil error")
			}
		},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil)
			c := e.NewContext(req, rec)
			tst(t, c)
		})
	}
}

func TestIsAdmin(t *testing.T) {
	is := is.New(t)
	is.True(!IsAdmin(&Claims{Roles: []Role{RoleDefault}}))
	is.True(IsAdmin(&Claims{Roles: []Role{RoleAdmin}}))
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
	if err := json.NewEncoder(&b).Encode(v); err != nil {
		panic(err)
	}
	return &b
}

type failingTokenConfig struct {
	TokenConfig
}

var errFailingTokenConfig = errors.New("this token config always fails")

func (ftc *failingTokenConfig) GetToken(r *http.Request) (string, error) {
	return "", errFailingTokenConfig
}

func generateRefreshToken(
	cfg TokenConfig,
	expiration time.Time,
	roles []Role,
	aud, iss string,
) string {
	c := Claims{
		ID:   mathrand.Int(),
		UUID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiration),
			Audience:  []string{aud},
			Issuer:    iss,
		},
	}
	tok := jwt.NewWithClaims(cfg.Type(), &c)
	token, err := tok.SignedString(cfg.Private())
	if err != nil {
		panic(err)
	}
	return token
}

func newTokenResp(conf TokenConfig, claims *Claims) (*TokenResponse, error) {
	tok := jwt.NewWithClaims(conf.Type(), claims)
	token, err := tok.SignedString(conf.Private())
	if err != nil {
		return nil, err
	}
	return &TokenResponse{
		Token:     token,
		Expires:   claims.ExpiresAt,
		TokenType: JWTScheme,
	}, nil
}
