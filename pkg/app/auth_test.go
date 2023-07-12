package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	mathrand "math/rand"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
	"golang.org/x/crypto/bcrypt"
	"gopkg.hrry.dev/homelab/pkg/auth"
	"gopkg.hrry.dev/homelab/pkg/internal/mocks/mockdb"
	"gopkg.hrry.dev/homelab/pkg/internal/mockutil"
)

func TestTokenHandler(t *testing.T) {
	defer silent()()
	var (
		ctx        = context.Background()
		loginQuery = func(db *mockdb.MockDB) *gomock.Call {
			return db.EXPECT().QueryContext(
				ctx,
				mockutil.HasPrefix(selectQueryHead),
				gomock.AssignableToTypeOf(""),
			)
		}
		loginScan = func(rows *mockdb.MockRows) *gomock.Call {
			return rows.EXPECT().Scan(
				gomock.AssignableToTypeOf(intptr),
				gomock.AssignableToTypeOf(&uuid.UUID{}),
				gomock.AssignableToTypeOf(strptr),   // username
				gomock.AssignableToTypeOf(strptr),   // email
				gomock.AssignableToTypeOf(bytesptr), // password hash
				gomock.AssignableToTypeOf(strptr),   // totp key
				gomock.Any(),                        // roles
				gomock.AssignableToTypeOf(&time.Time{}),
				gomock.AssignableToTypeOf(&time.Time{}),
			)
		}
	)

	type table struct {
		cfg   auth.TokenConfig
		body  map[string]interface{}
		errs  []error
		query url.Values
		prep  func(db *mockdb.MockDB, rows *mockdb.MockRows)
	}

	for i, tt := range []table{
		{
			cfg:  auth.GenEdDSATokenConfig(),
			body: nil,
			errs: []error{ErrEmptyPassword, echo.ErrNotFound},
		},
		{
			cfg:  auth.GenEdDSATokenConfig(),
			body: map[string]interface{}{"password": "1234"},
			errs: []error{ErrUserNotFound, echo.ErrNotFound},
		},
		{
			cfg:  auth.GenerateECDSATokenConfig(),
			body: map[string]interface{}{"email": "joe@joe.com", "password": "im the real joe"},
			errs: []error{sql.ErrConnDone, echo.ErrNotFound},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Return(rows, nil)
				rows.EXPECT().Next().Return(true)
				loginScan(rows).Return(sql.ErrConnDone)
				rows.EXPECT().Close().Return(nil)
			},
		},
		{
			cfg:  auth.GenerateECDSATokenConfig(),
			body: map[string]interface{}{"email": "joe@joe.com", "password": "im the real joe"},
			errs: []error{ErrUserNotFound, echo.ErrNotFound},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Return(rows, nil)
				rows.EXPECT().Next().Return(false)
				rows.EXPECT().Err().Return(ErrUserNotFound)
				rows.EXPECT().Close().Return(nil)
			},
		},
		{
			cfg:  auth.GenerateECDSATokenConfig(),
			body: map[string]interface{}{"email": "joe@joe.com", "password": "im the real joe"},
			errs: []error{sql.ErrNoRows, echo.ErrNotFound},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Return(rows, sql.ErrNoRows)
			},
		},
		{
			cfg:  auth.GenEdDSATokenConfig(),
			body: map[string]interface{}{"username": "tester", "password": "asdfasdf"},
			errs: []error{sql.ErrNoRows, echo.ErrNotFound},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Times(1).Return(rows, nil)
				rows.EXPECT().Next().Return(false)
				rows.EXPECT().Err().Return(nil)
				rows.EXPECT().Close().Return(nil)
			},
		},
		{
			cfg:  auth.GenerateECDSATokenConfig(),
			body: map[string]interface{}{"email": "joe@joe.com", "password": "im the real joe"},
			errs: []error{ErrWrongPassword, echo.ErrNotFound},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Return(rows, nil)
				rows.EXPECT().Next().Return(true)
				loginScan(rows).Return(nil)
				rows.EXPECT().Close().Return(nil)
			},
		},
		{
			cfg:   auth.GenerateECDSATokenConfig(),
			body:  map[string]interface{}{"username": "tester", "password": "asdfasdf"},
			query: url.Values{"cookie": {"true"}},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				loginQuery(db).Times(1).Return(rows, nil)
				rows.EXPECT().Next().Return(true)
				loginScan(rows).Times(1).Do(func(v ...interface{}) {
					pw := v[4].(*[]byte)
					var e error
					*pw, e = bcrypt.GenerateFromPassword([]byte("asdfasdf"), hashCost)
					if e != nil {
						t.Fatal(e)
					}
				}).Return(nil)
				rows.EXPECT().Close().Return(nil)
			},
		},
		{
			cfg:   auth.GenEdDSATokenConfig(),
			errs:  []error{echo.ErrBadRequest},
			query: url.Values{"cookie": {"notaboolean"}},
		},
	} {
		if tt.prep == nil {
			tt.prep = func(*mockdb.MockDB, *mockdb.MockRows) {}
		}
		t.Run(fmt.Sprintf("TestTokenHandler_%d", i), func(t *testing.T) {
			var (
				is   = is.New(t)
				e    = echo.New()
				ctrl = gomock.NewController(t)
				db   = mockdb.NewMockDB(ctrl)
				rows = mockdb.NewMockRows(ctrl)
				rec  = httptest.NewRecorder()
				req  = httptest.NewRequest("POST", "/", body(tt.body)).WithContext(ctx)
			)
			defer ctrl.Finish()
			req.Header.Set(echo.HeaderContentType, "application/json")
			req.URL.RawQuery = tt.query.Encode()

			service := TokenService{
				Config: tt.cfg,
				Users:  NewUserStore(db),
				Tokens: auth.NewInMemoryTokenStore(time.Minute),
			}
			tt.prep(db, rows)
			c := e.NewContext(req, rec)
			err := service.Token(c)
			checkErrs(t, tt.errs, err)

			if len(tt.errs) > 0 {
				return
			}

			resp := rec.Result()
			defer resp.Body.Close()
			var (
				tok     *auth.TokenResponse
				cookies = resp.Cookies()
			)
			is.NoErr(json.NewDecoder(resp.Body).Decode(&tok))
			is.True(len(tok.Token) > 1)
			is.True(len(tok.RefreshToken) > 1)
			is.Equal(1, len(cookies))
			is.Equal("/", cookies[0].Path)
			is.Equal(cookies[0].Value, tok.Token)
			is.Equal(cookies[0].Expires, tok.Expires.Time.UTC())
			claims := auth.GetClaims(c)
			is.True(claims != nil)
		})
	}
}

func TestRefreshTokenHandler_Err(t *testing.T) {
	type table struct {
		prep func(claims *auth.Claims, v url.Values)
	}
	for i, tt := range []table{
		{prep: func(claims *auth.Claims, v url.Values) { claims.Audience = []string{"not correct aud"} }},
		{prep: func(claims *auth.Claims, v url.Values) {
			n := time.Now()
			claims.ExpiresAt = jwt.NewNumericDate(time.Date(
				n.Year(), n.Month(), n.Day(), n.Hour()-5,
				n.Minute(), n.Second(), n.Nanosecond(), n.Location(),
			))
		}},
		{prep: func(claims *auth.Claims, v url.Values) {
			v.Add("cookie", "not-a-boolean")
		}},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			tokenCfg := auth.GenEdDSATokenConfig()
			claims := &auth.Claims{
				ID:   mathrand.Int(),
				UUID: uuid.New(),
				RegisteredClaims: jwt.RegisteredClaims{
					IssuedAt:  jwt.NewNumericDate(time.Now()),
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(auth.Expiration)),
					Audience:  []string{"refresh"},
					Issuer:    auth.Issuer,
				},
			}
			params := url.Values{}
			userClaims := *claims
			userClaims.Audience = []string{auth.TokenAudience}
			if tt.prep != nil {
				tt.prep(claims, params)
			}
			refreshToken, err := jwt.NewWithClaims(tokenCfg.Type(), claims).SignedString(tokenCfg.Private())
			is.NoErr(err)
			store := auth.NewInMemoryTokenStore(time.Minute)
			is.NoErr(store.Set(context.Background(), claims.ID, refreshToken))
			e := echo.New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("POST", "/api/refresh", body(map[string]interface{}{"refresh_token": refreshToken}))
			req.URL.RawQuery = params.Encode()
			req.Header.Set("Content-Type", "application/json")
			c := e.NewContext(req, rec)
			service := TokenService{
				Tokens: store,
				Config: tokenCfg,
			}
			err = service.Refresh(c)
			if err == nil {
				t.Fatal("expected an error got nil")
			}
		})
	}
}

func TestRefreshTokenHandler(t *testing.T) {
	is := is.New(t)
	tokenCfg := auth.GenEdDSATokenConfig()
	claims := &auth.Claims{
		ID:   mathrand.Int(),
		UUID: uuid.New(),
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(auth.Expiration)),
			Audience:  []string{"refresh"},
			Issuer:    auth.Issuer,
		},
	}
	refreshToken, err := jwt.NewWithClaims(tokenCfg.Type(), claims).SignedString(tokenCfg.Private())
	is.NoErr(err)
	store := auth.NewInMemoryTokenStore(time.Minute)
	is.NoErr(store.Set(context.Background(), claims.ID, refreshToken))
	e := echo.New()
	req := httptest.NewRequest(
		"POST", "/api/refresh?cookie=true",
		body(map[string]interface{}{"refresh_token": refreshToken}),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	service := TokenService{
		Config: tokenCfg,
		Tokens: store,
	}
	err = service.Refresh(c)
	is.NoErr(err)
	res := rec.Result()
	defer res.Body.Close()
	is.Equal(res.StatusCode, 200)
	is.Equal(1, len(res.Cookies()))
	cookie := res.Cookies()[0]
	is.Equal(cookie.Name, tokenKey)
	is.Equal(cookie.Path, "/")
	var tok auth.TokenResponse
	is.NoErr(json.NewDecoder(res.Body).Decode(&tok))
	is.True(1 < len(tok.Token))
}
