package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/lib/pq"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"harrybrown.com/internal/mocks/mockdb"
	"harrybrown.com/internal/mocks/mockredis"
	"harrybrown.com/internal/mockutil"
	"harrybrown.com/pkg/auth"
)

type testPath struct {
	p  string
	id string
	t  *testing.T
}

func (tp *testPath) Path(id string) string {
	return filepath.Join("/", tp.p, id)
}

func (tp *testPath) GetID(req *http.Request) string {
	list := strings.Split(req.URL.Path, string(filepath.Separator))
	if len(list) >= 3 {
		return list[2]
	}
	return ""
}

func TestInviteCreate(t *testing.T) {
	defer silent()()
	type table struct {
		id       string
		claims   *auth.Claims
		body     CreateInviteRequest
		expected error
		internal error
	}

	for i, tt := range []table{
		{
			// Admin can change timeout
			id:     "1",
			claims: &auth.Claims{Roles: []auth.Role{auth.RoleAdmin}},
			body:   CreateInviteRequest{Timeout: time.Minute},
		},
		{
			// Admin can change ttl
			id:     "12",
			claims: &auth.Claims{Roles: []auth.Role{auth.RoleAdmin}},
			body:   CreateInviteRequest{TTL: 64},
		},
		{
			// Admin can change roles
			id:     "123",
			claims: &auth.Claims{Roles: []auth.Role{auth.RoleAdmin}},
			body:   CreateInviteRequest{Roles: []string{"some_role"}},
		},
		{
			// Regular user not allowed to change timeout
			id:       "1234",
			claims:   &auth.Claims{Roles: []auth.Role{auth.RoleDefault}},
			body:     CreateInviteRequest{Timeout: time.Minute},
			expected: echo.ErrUnauthorized,
			internal: auth.ErrAdminRequired,
		},
		{
			// Regular user not allowed to change ttl
			id:       "12345",
			claims:   &auth.Claims{Roles: []auth.Role{auth.RoleDefault}},
			body:     CreateInviteRequest{TTL: 1000},
			expected: echo.ErrUnauthorized,
			internal: auth.ErrAdminRequired,
		},
		{
			// Regular user not allowed to change roles
			id:       "123456",
			claims:   &auth.Claims{Roles: []auth.Role{auth.RoleDefault}},
			body:     CreateInviteRequest{Roles: []string{"admin"}},
			expected: echo.ErrUnauthorized,
			internal: auth.ErrAdminRequired,
		},
		{
			id:     "2",
			claims: &auth.Claims{Roles: []auth.Role{auth.RoleAdmin}},
			body:   CreateInviteRequest{Email: "t@t.com"},
		},
		{
			id:     "3",
			claims: &auth.Claims{Roles: []auth.Role{auth.RoleDefault}},
			body:   CreateInviteRequest{Email: "t@t.com"},
		},
		{
			id:       "4",
			claims:   nil,
			body:     CreateInviteRequest{},
			expected: echo.ErrUnauthorized,
			internal: auth.ErrNoClaims,
		},
		{
			id:       "5",
			claims:   &auth.Claims{Roles: []auth.Role{auth.RoleAdmin}},
			body:     CreateInviteRequest{Timeout: -100},
			expected: ErrInvalidTimeout,
			internal: nil,
		},
	} {
		tt := tt
		i := i
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			tm := time.Now()
			now = func() time.Time { return tm }
			invites := Invitations{
				Path: &testPath{p: "invite", id: tt.id, t: t},
				// RDB:  rdb,
				// Encoder: &staticEncoding{id: tt.id},
				store: &InviteSessionStore{
					RDB:    rdb,
					Prefix: "invite",
				},
			}
			defer ctrl.Finish()
			if tt.claims != nil {
				tt.claims.UUID = uuid.New()
				tt.claims.ID = i
			}
			req := httptest.NewRequest("POST", invites.Path.Path(tt.id), body(tt.body)).WithContext(ctx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set(auth.ClaimsContextKey, tt.claims)

			// Find expected values
			if tt.body.Timeout == 0 {
				tt.body.Timeout = defaultInviteTimeout
			}
			if tt.body.TTL == 0 {
				tt.body.TTL = defaultInviteTTL
			}
			roles := make([]auth.Role, len(tt.body.Roles))
			for i, r := range tt.body.Roles {
				roles[i] = auth.Role(r)
			}
			if tt.expected == nil {
				expires := tm.Add(tt.body.Timeout).UnixMilli()
				expectedSession, err := json.Marshal(&inviteSession{
					CreatedBy: tt.claims.UUID,
					TTL:       tt.body.TTL,
					ExpiresAt: expires,
					Email:     tt.body.Email,
					Roles:     roles,
				})
				is.NoErr(err)
				rdb.EXPECT().
					Set(ctx, mockutil.HasPrefix("invite:"), expectedSession, tt.body.Timeout).
					Return(redis.NewStatusResult("", nil))
			}
			err := invites.Create()(c)
			is.True(errors.Is(err, tt.expected))
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected == nil {
				var resp invite
				is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))
				is.True(len(resp.Path) > 0)
			}
		})
	}

	is := is.New(t)
	req := httptest.NewRequest("POST", "/", strings.NewReader("{")) // bad json
	c := echo.New().NewContext(req, httptest.NewRecorder())
	c.Set(auth.ClaimsContextKey, &auth.Claims{})
	err := (&Invitations{}).Create()(c)
	is.True(err != nil)                // Expecting an error
	is.Equal(io.ErrUnexpectedEOF, err) // Bad json should return unexpected EOF
}

func TestInviteAccept(t *testing.T) {
	type table struct {
		expected error
		internal error
		session  *inviteSession
		mocks    func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession)
	}
	for i, tt := range []table{
		{
			session: &inviteSession{TTL: 5, Email: "t@t.io", ExpiresAt: 12},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				mockSessionGet(t, rdb, gomock.Eq("invite:12345"), s).Times(1)
			},
		},
		{
			// Expired TTL of 0
			session: &inviteSession{
				TTL:       0,
				Email:     "test@x.io",
				ExpiresAt: 1001,
				CreatedBy: uuid.New(),
			},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				mockSessionGet(t, rdb, gomock.Eq("invite:12345"), s).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.Eq("invite:12345")).Return(redis.NewIntResult(0, nil))
			},
			expected: echo.ErrNotFound,
			internal: ErrInviteTTL,
		},
		{
			expected: echo.ErrForbidden,
			session: &inviteSession{
				TTL:       -1,
				Email:     "test@x.io",
				ExpiresAt: -10,
				CreatedBy: uuid.New(),
			},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				mockSessionGet(t, rdb, gomock.Eq("invite:12345"), s).Times(1)
			},
		},
		{
			// Always good TTL of -1
			session: &inviteSession{
				TTL:       -1,
				Email:     "test@x.io",
				ExpiresAt: 1001,
				CreatedBy: uuid.New(),
			},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				mockSessionGet(t, rdb, gomock.Eq("invite:12345"), s).Times(1)
			},
		},
		{
			// Session not found
			expected: echo.ErrNotFound,
			internal: redis.Nil,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				rdb.EXPECT().Get(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.Eq("invite:12345"),
				).Return(redis.NewStringResult("", redis.Nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			invites := Invitations{
				Path: &testPath{p: "invite"},
				// RDB:  rdb,
				// Encoder: base64.RawURLEncoding,
				store: &InviteSessionStore{RDB: rdb, Prefix: "invite"},
			}
			defer ctrl.Finish()

			id := "12345"
			req := httptest.NewRequest("GET", invites.Path.Path(id), nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			template := `{"email":"{{ .Email }}","expires":{{ .ExpiresAt }}}`
			handler := invites.Accept([]byte(template), "text/html")

			if tt.mocks != nil {
				tt.mocks(t, rdb, tt.session)
			}
			err := handler(c)
			is.True(errors.Is(tt.expected, err))
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected != nil {
				return
			}

			data := struct {
				Email   string `json:"email"`
				Expires int64  `json:"expires"`
			}{}
			is.NoErr(json.NewDecoder(rec.Body).Decode(&data))
			is.Equal(data.Email, tt.session.Email)
			is.Equal(data.Expires, tt.session.ExpiresAt)
		})
	}
}

func TestInviteSignUp(t *testing.T) {
	type mocks struct {
		rdb  *mockredis.MockCmdable
		db   *mockdb.MockDB
		rows *mockdb.MockRows
	}
	type table struct {
		name               string
		expected, internal error
		session            *inviteSession
		mocks              func(t *testing.T, tt *table, mocks *mocks)
		login              *Login
	}

	mockSessionUpdate := func(t *testing.T, rdb *mockredis.MockCmdable, session *inviteSession) *gomock.Call {
		s := *session
		s.TTL--
		raw, err := json.Marshal(s)
		if err != nil {
			t.Fatal(err)
		}
		return rdb.EXPECT().Set(
			context.Background(),
			"invite:444",
			raw,
			time.Duration(redis.KeepTTL),
		)
	}
	randomError := errors.New("this is some random error")

	for i, tt := range []table{
		{
			name:     "session not found",
			expected: echo.ErrNotFound,
			internal: redis.Nil,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mocks.rdb.EXPECT().Get(context.Background(), "invite:444").Return(redis.NewStringResult("", redis.Nil))
			},
		},
		{
			name:     "expired ttl",
			session:  &inviteSession{TTL: 0},
			expected: echo.ErrForbidden,
			internal: ErrInviteTTL,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session).Times(1)
				mocks.rdb.EXPECT().Del(
					context.Background(),
					gomock.Eq("invite:444"),
				).Return(redis.NewIntResult(0, nil))
			},
		},
		{
			name:     "Fail to update session with new ttl",
			session:  &inviteSession{TTL: 10},
			expected: echo.ErrNotFound,
			internal: redis.Nil,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", redis.Nil))
			},
		},
		{
			name:     "Fail to parse request body",
			session:  &inviteSession{TTL: 64},
			expected: echo.ErrBadRequest,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "no email",
			session:  &inviteSession{TTL: 12},
			login:    &Login{Password: "yeet", Username: "abc"},
			expected: ErrEmptyLogin,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "no password",
			session:  &inviteSession{TTL: -1},
			login:    &Login{Email: "yeet@yeet.com", Username: "abc"},
			expected: ErrEmptyLogin,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
			},
		},
		{
			name:     "email missmatch",
			session:  &inviteSession{TTL: 55, Email: "what@theheck.org"},
			login:    &Login{Email: "what@not_theheck.org", Password: "password1"},
			expected: ErrInviteEmailMissmatch,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "failed to create user",
			session:  &inviteSession{TTL: -1},
			login:    &Login{Email: "a@a.it", Password: "123", Username: "test-user"},
			expected: echo.ErrInternalServerError, internal: randomError,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session)
				mocks.db.EXPECT().QueryContext(
					context.Background(), createUserQuery,
					gomock.Any(),
					tt.login.Username,
					tt.login.Email,
					gomock.Any(), gomock.Any(), gomock.Any(),
				).Return(nil, randomError)
			},
		},
		{
			name:    "success",
			session: &inviteSession{TTL: -1, Roles: []auth.Role{auth.RoleAdmin, auth.RoleDefault}},
			login:   &Login{Email: "a@a.it", Password: "123", Username: "test-user"},
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				ctx := context.Background()
				gomock.InOrder(
					mockSessionGet(t, mocks.rdb, gomock.Eq("invite:444"), tt.session),
					mocks.db.EXPECT().QueryContext(
						ctx, createUserQuery,
						gomock.Any(), // uuid
						tt.login.Username,
						tt.login.Email,
						gomock.Any(), // password hash
						pq.Array(tt.session.Roles),
						gomock.Any(), // totp secret
					).Return(mocks.rows, nil),
					mocks.rows.EXPECT().Next().Return(true),
					mocks.rows.EXPECT().Scan(
						gomock.AssignableToTypeOf(intptr),
						gomock.AssignableToTypeOf(&time.Time{}),
						gomock.AssignableToTypeOf(&time.Time{}),
					).Return(nil),
					mocks.rows.EXPECT().Close().Return(nil),
					mocks.rdb.EXPECT().Del(ctx, gomock.Eq("invite:444")).Return(redis.NewIntResult(1, nil)),
				)
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			ctx := context.Background()
			is := is.New(t)
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			db := mockdb.NewMockDB(ctrl)
			rows := mockdb.NewMockRows(ctrl)
			invites := Invitations{
				Path: &testPath{p: "invite"},
				// RDB:  rdb,
				// Encoder: base64.RawURLEncoding,
				store: &InviteSessionStore{
					RDB:    rdb,
					Prefix: "invite",
				},
			}
			defer ctrl.Finish()

			req := httptest.NewRequest("POST", "/invite/444", body(tt.login)).WithContext(ctx)
			if tt.login != nil {
				req.Header.Set("Content-Type", "application/json")
			}
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			if tt.mocks != nil {
				tt.mocks(t, &tt, &mocks{rdb: rdb, db: db, rows: rows})
			}
			err := invites.SignUp(NewUserStore(db))(c)
			is.True(errors.Is(tt.expected, err))
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected != nil {
				return
			}
			// TODO Check output
			// is.Equal(rec.Code, http.StatusPermanentRedirect)
			// is.Equal(rec.Header().Get("location"), "/")
			is.Equal(rec.Code, http.StatusOK)
		})
	}
}

func TestInviteDelete(t *testing.T) {
	type table struct {
		name               string
		expected, internal error
		id                 string
		claims             *auth.Claims
		session            *inviteSession
		mocks              func(t *testing.T, rdb *mockredis.MockCmdable, tt *table)
	}
	for i, tt := range []table{
		{
			name:     "no claims",
			id:       "oneTwoThreeFour",
			claims:   nil,
			expected: echo.ErrUnauthorized,
		},
		{
			name:     "fail to get session",
			id:       "qwerty",
			claims:   &auth.Claims{UUID: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			session:  &inviteSession{CreatedBy: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			expected: echo.ErrNotFound,
			internal: redis.Nil,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				rdb.EXPECT().Get(
					context.Background(),
					"invite:qwerty",
				).Return(redis.NewStringResult("", redis.Nil))
			},
		},
		{
			name:     "wrong uuid",
			id:       "123",
			claims:   &auth.Claims{UUID: uuid.MustParse("11111111-4d00-458d-927d-d4416d10c68f")},
			session:  &inviteSession{CreatedBy: uuid.MustParse("22222222-4d00-458d-927d-d4416d10c68f")},
			expected: echo.ErrForbidden,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				mockSessionGet(t, rdb, gomock.Eq("invite:123"), tt.session).Times(1)
			},
		},
		{
			name:     "fail to delete session",
			id:       "123321",
			claims:   &auth.Claims{UUID: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			session:  &inviteSession{CreatedBy: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			expected: echo.ErrInternalServerError,
			internal: redis.ErrClosed,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				mockSessionGet(t, rdb, gomock.Eq("invite:123321"), tt.session).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.Eq("invite:123321")).
					Return(redis.NewIntResult(0, redis.ErrClosed))
			},
		},
		{
			name:    "success",
			id:      "123",
			claims:  &auth.Claims{UUID: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			session: &inviteSession{CreatedBy: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				mockSessionGet(t, rdb, gomock.Eq("invite:123"), tt.session).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.Eq("invite:123")).Return(redis.NewIntResult(0, nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			ctx := context.Background()
			is := is.New(t)
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			invites := Invitations{
				Path: &testPath{p: "invite"},
				// RDB:  rdb,
				// Encoder: base64.RawURLEncoding,
				store: &InviteSessionStore{RDB: rdb, Prefix: "invite"},
			}
			defer ctrl.Finish()

			req := httptest.NewRequest("DELETE", fmt.Sprintf("/invite/%s", tt.id), nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set(auth.ClaimsContextKey, tt.claims)
			c.SetParamNames("id")
			c.SetParamValues(tt.id)

			if tt.mocks != nil {
				tt.mocks(t, rdb, &tt)
			}
			err := invites.Delete()(c)
			is.True(errors.Is(tt.expected, err)) // should have expected error
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected != nil {
				return
			}
		})
	}
}

func TestInviteList(t *testing.T) {
	type table struct {
		name               string
		expected, internal error
		claims             *auth.Claims
		inviteList         *inviteList
		sessions           []inviteSession
		mock               func(rdb *mockredis.MockCmdable, tt *table, sessions []inviteSession)
	}
	for i, tt := range []table{
		{
			name:     "no claims",
			expected: echo.ErrUnauthorized,
		},
		{
			name:     "failed getting keys",
			expected: echo.ErrInternalServerError,
			internal: redis.Nil,
			claims:   new(auth.Claims),
			mock: func(rdb *mockredis.MockCmdable, tt *table, sessions []inviteSession) {
				rdb.EXPECT().Keys(context.Background(), "invite:*").
					Return(redis.NewStringSliceResult(nil, redis.Nil))
			},
		},
		{
			name:       "no sessions",
			inviteList: &inviteList{},
			claims:     &auth.Claims{},
			mock: func(rdb *mockredis.MockCmdable, tt *table, sessions []inviteSession) {
				rdb.EXPECT().Keys(context.Background(), "invite:*").
					Return(redis.NewStringSliceResult([]string{}, nil))
			},
		},
		{
			name: "list as admin",
			claims: &auth.Claims{
				UUID:  uuid.MustParse("e5ccb6f1-816f-4d67-821b-64be606af220"),
				Roles: []auth.Role{auth.RoleAdmin},
			},
			inviteList: &inviteList{
				Invites: []invite{
					{
						Path:      "/invite/1",
						CreatedBy: uuid.MustParse("aabbccdd-816f-4d67-821b-64be606af220"),
						ExpiresAt: time.UnixMilli(100000),
					},
					{
						Path:      "/invite/2",
						CreatedBy: uuid.MustParse("eeff1122-816f-4d67-821b-64be606af220"),
						ExpiresAt: time.UnixMilli(1000000),
					},
				},
			},
			sessions: []inviteSession{
				{
					CreatedBy: uuid.MustParse("aabbccdd-816f-4d67-821b-64be606af220"),
					ExpiresAt: 100000,
				},
				{
					CreatedBy: uuid.MustParse("eeff1122-816f-4d67-821b-64be606af220"),
					ExpiresAt: 1000000,
				},
			},
			mock: func(rdb *mockredis.MockCmdable, tt *table, sessions []inviteSession) {
				ctx := context.Background()
				keys := []string{
					"invite:1",
					"invite:2",
				}
				rdb.EXPECT().Keys(ctx, "invite:*").
					Return(redis.NewStringSliceResult(keys, nil))
				raw := make([]interface{}, len(tt.sessions))
				for i, s := range tt.sessions {
					b, err := json.Marshal(s)
					if err != nil {
						panic(err)
					}
					raw[i] = string(b)
				}
				rdb.EXPECT().MGet(ctx, keys[0], keys[1]).
					Return(redis.NewSliceResult(raw, nil))
			},
		},
		{
			name: "list as not admin",
			claims: &auth.Claims{
				UUID:  uuid.MustParse("e5ccb6f1-816f-4d67-821b-64be606af220"),
				Roles: []auth.Role{auth.RoleDefault},
			},
			inviteList: &inviteList{
				Invites: []invite{
					{
						Path:      "/invite/3",
						CreatedBy: uuid.MustParse("e5ccb6f1-816f-4d67-821b-64be606af220"),
						ExpiresAt: time.UnixMilli(123),
					},
				},
			},
			sessions: []inviteSession{
				{
					CreatedBy: uuid.MustParse("aabbccdd-816f-4d67-821b-64be606af220"),
					ExpiresAt: 1,
				},
				{
					CreatedBy: uuid.MustParse("eeff1122-816f-4d67-821b-64be606af220"),
					ExpiresAt: 1,
				},
				{
					CreatedBy: uuid.MustParse("e5ccb6f1-816f-4d67-821b-64be606af220"),
					ExpiresAt: 123,
				},
			},
			mock: func(rdb *mockredis.MockCmdable, tt *table, sessions []inviteSession) {
				ctx := context.Background()
				keys := []string{"invite:1", "invite:2", "invite:3"}
				rdb.EXPECT().Keys(ctx, "invite:*").
					Return(redis.NewStringSliceResult(keys, nil))
				raw := make([]interface{}, len(tt.sessions))
				for i, s := range tt.sessions {
					b, err := json.Marshal(s)
					if err != nil {
						panic(err)
					}
					raw[i] = string(b)
				}
				rdb.EXPECT().MGet(ctx, keys[0], keys[1], keys[2]).
					Return(redis.NewSliceResult(raw, nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			ctx := context.Background()
			is := is.New(t)
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			invites := Invitations{
				Path: &testPath{p: "invite"},
				// RDB:  rdb,
				// Encoder: base64.RawURLEncoding,
				store: &InviteSessionStore{RDB: rdb, Prefix: "invite"},
			}
			defer ctrl.Finish()

			req := httptest.NewRequest("GET", "/invite/list", nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set(auth.ClaimsContextKey, tt.claims)
			if tt.mock != nil {
				tt.mock(rdb, &tt, nil)
			}
			err := invites.List()(c)
			is.True(errors.Is(tt.expected, err))
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected != nil {
				return
			}
			list := inviteList{}
			err = json.Unmarshal(rec.Body.Bytes(), &list)
			is.NoErr(err)
			is.Equal(len(list.Invites), len(tt.inviteList.Invites))
			for i := range list.Invites {
				is.Equal(list.Invites[i].CreatedBy, tt.inviteList.Invites[i].CreatedBy)
				is.Equal(list.Invites[i].Path, tt.inviteList.Invites[i].Path)
				is.True(list.Invites[i].ExpiresAt.Equal(tt.inviteList.Invites[i].ExpiresAt))
			}
		})
	}
}

func TestInviteSessionStore_Get(t *testing.T) {
	logger.SetOutput(io.Discard)
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rdb := mockredis.NewMockCmdable(ctrl)
	s := InviteSessionStore{RDB: rdb, Prefix: "inv"}
	ctx := context.Background()

	// Decrement TTL
	expectedSession := inviteSession{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 10}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	updated := inviteSession{CreatedBy: expectedSession.CreatedBy, Roles: expectedSession.Roles, TTL: 9}
	mockSessionSet(t, rdb, gomock.Eq("inv:123"), gomock.Eq(time.Duration(redis.KeepTTL)), &updated).Return(redis.NewStatusResult("", nil))
	session, err := s.Get(ctx, "123")
	is.NoErr(err)
	is.Equal(*session, updated)
	is.Equal(session.TTL, expectedSession.TTL-1)

	// Delete because of expired TTL
	expectedSession = inviteSession{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 0}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	rdb.EXPECT().Del(ctx, "inv:123").Return(redis.NewIntResult(0, nil))
	session, err = s.Get(ctx, "123")
	is.Equal(err, ErrInviteTTL)
	is.Equal(session, nil)

	// Delete because of expired TTL: failed delete
	expectedSession = inviteSession{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 0}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	rdb.EXPECT().Del(ctx, "inv:123").Return(redis.NewIntResult(0, errors.New("this is some random error")))
	session, err = s.Get(ctx, "123")
	is.Equal(err, ErrInviteTTL)
	is.Equal(session, nil)

	// Ignore negative TTL
	expectedSession = inviteSession{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: -1}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	session, err = s.Get(ctx, "123")
	is.NoErr(err)
	is.Equal(*session, expectedSession)
}

func TestInviteSessionStore_Create(t *testing.T) {
	type table struct {
		name     string
		req      *CreateInviteRequest
		uuid     uuid.UUID
		expected error
		mock     func(t *testing.T, tt *table, rd *mockredis.MockCmdable)
	}
	tmpErr := errors.New("testing error")
	for i, tt := range []table{
		{
			name:     "failed redis set",
			req:      &CreateInviteRequest{},
			expected: tmpErr,
			mock: func(t *testing.T, tt *table, rd *mockredis.MockCmdable) {
				now = func() time.Time { return time.Unix(100, 0) }
				mockSessionSet(
					t, rd, mockutil.HasPrefix("iss_create:"),
					gomock.Eq(defaultInviteTimeout),
					&inviteSession{TTL: defaultInviteTTL, ExpiresAt: now().Add(defaultInviteTimeout).UnixMilli(), CreatedBy: tt.uuid},
				).Return(redis.NewStatusResult("", tmpErr))
			},
		},
		{
			name: "default values",
			req:  &CreateInviteRequest{},
			mock: func(t *testing.T, tt *table, rd *mockredis.MockCmdable) {
				now = func() time.Time { return time.Unix(100, 0) }
				mockSessionSet(
					t, rd, mockutil.HasPrefix("iss_create:"),
					gomock.Eq(defaultInviteTimeout),
					&inviteSession{TTL: defaultInviteTTL, ExpiresAt: now().Add(defaultInviteTimeout).UnixMilli(), CreatedBy: tt.uuid},
				).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name: "custom request values",
			req:  &CreateInviteRequest{TTL: 53, Timeout: time.Hour * 9},
			mock: func(t *testing.T, tt *table, rd *mockredis.MockCmdable) {
				now = func() time.Time { return time.Unix(1010, 0) }
				mockSessionSet(
					t, rd, mockutil.HasPrefix("iss_create:"),
					gomock.Eq(time.Hour*9),
					&inviteSession{TTL: 53, ExpiresAt: now().Add(time.Hour * 9).UnixMilli(), CreatedBy: tt.uuid},
				).Return(redis.NewStatusResult("", nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			is := is.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			rd := mockredis.NewMockCmdable(ctrl)
			store := InviteSessionStore{Prefix: "iss_create", RDB: rd}
			ctx := context.Background()
			tt.mock(t, &tt, rd)
			session, id, err := store.Create(ctx, tt.uuid, tt.req)
			is.True(errors.Is(err, tt.expected))
			if tt.expected != nil {
				return
			}
			is.True(len(id) > 0)
			is.True(session != nil)
		})
	}
}

func TestInviteSessionStore_Del(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rd := mockredis.NewMockCmdable(ctrl)
	ctx := context.Background()
	store := InviteSessionStore{RDB: rd, Prefix: "i"}

	uid := uuid.New()

	mockSessionGet(t, rd, gomock.Eq("i:abc"), &inviteSession{CreatedBy: uid})
	rd.EXPECT().Del(ctx, "i:abc").Return(redis.NewIntResult(0, nil))
	err := store.OwnerDel(ctx, "abc", uid)
	is.NoErr(err)

	mockSessionGet(t, rd, gomock.Eq("i:abc"), &inviteSession{CreatedBy: uid})
	err = store.OwnerDel(ctx, "abc", uuid.New())
	is.True(errors.Is(err, ErrSessionOwnership))
}

func TestInviteSessionStore_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var err error
	is := is.New(t)
	rdb := mockredis.NewMockCmdable(ctrl)
	s := InviteSessionStore{RDB: rdb, Prefix: "inv"}
	ctx := context.Background()
	keys := []string{"inv:1", "inv:2", "inv:3"}
	expected := []*inviteSession{
		{id: "1", CreatedBy: uuid.New(), ExpiresAt: 1000, Roles: []auth.Role{auth.RoleAdmin}},
		{id: "2", CreatedBy: uuid.New(), ExpiresAt: 1001, Roles: []auth.Role{auth.RoleDefault}},
		{id: "3", CreatedBy: uuid.New(), ExpiresAt: 1002, Roles: []auth.Role{auth.RoleFamily}},
	}
	expectedJSON := make([]interface{}, len(expected))
	for i, exp := range expected {
		raw, err := json.Marshal(exp)
		is.NoErr(err)
		expectedJSON[i] = string(raw)
	}

	rdb.EXPECT().Keys(ctx, "inv:*").Return(redis.NewStringSliceResult(keys, nil))
	rdb.EXPECT().MGet(ctx, keys).Return(redis.NewSliceResult(expectedJSON, nil))
	sessions, err := s.List(ctx)
	is.NoErr(err)
	is.Equal(len(sessions), len(expected))
	is.Equal(sessions, expected)

	someErr := errors.New("some error")
	rdb.EXPECT().Keys(ctx, "inv:*").Return(redis.NewStringSliceResult(keys, nil))
	rdb.EXPECT().MGet(ctx, keys).Return(redis.NewSliceResult(nil, someErr))
	_, err = s.List(ctx)
	is.Equal(err, someErr)

	rdb.EXPECT().Keys(ctx, "inv:*").Return(redis.NewStringSliceResult([]string{}, nil))
	sessions, err = s.List(ctx)
	is.NoErr(err)
	is.Equal(len(sessions), 0)

	rdb.EXPECT().Keys(ctx, "inv:*").Return(redis.NewStringSliceResult(nil, someErr))
	_, err = s.List(ctx)
	is.Equal(err, someErr)
}

func mockSessionGet(t *testing.T, rdb *mockredis.MockCmdable, key gomock.Matcher, s *inviteSession) *gomock.Call {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return rdb.EXPECT().Get(
		context.Background(), key,
	).Return(redis.NewStringResult(string(raw), nil))
}

func mockSessionSet(
	t *testing.T,
	rd *mockredis.MockCmdable,
	key gomock.Matcher,
	timeout gomock.Matcher,
	s *inviteSession,
) *gomock.Call {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return rd.EXPECT().Set(context.Background(), key, raw, timeout)
}
