package app

import (
	"context"
	"encoding/base64"
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
	"harrybrown.com/pkg/auth"
)

type testPath struct {
	p  string
	id string
	t  *testing.T
}

func (tp *testPath) Path(id string) string {
	if tp.t != nil && id != tp.id {
		tp.t.Helper()
		tp.t.Errorf("wrong id: got %s, want %s", id, tp.id)
	}
	return filepath.Join("/", tp.p, id)
}

func (tp *testPath) GetID(req *http.Request) string {
	list := strings.Split(req.URL.Path, string(filepath.Separator))
	if len(list) >= 3 {
		return list[2]
	}
	return ""
}

type staticEncoding struct {
	id string
}

func (se *staticEncoding) EncodeToString(_ []byte) string {
	return se.id
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
	} {
		tt := tt
		i := i
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			ctx := context.Background()
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			now := time.Now()
			invites := Invitations{
				Path:    &testPath{p: "invite", id: tt.id, t: t},
				RDB:     rdb,
				Encoder: &staticEncoding{id: tt.id},
				Now:     func() time.Time { return now },
			}
			defer ctrl.Finish()
			if tt.claims != nil {
				tt.claims.UUID = uuid.New()
				tt.claims.ID = i
				// fmt.Println(tt.claims.UUID)
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
				expires := now.Add(tt.body.Timeout).UnixMilli()
				expectedSession, err := json.Marshal(&inviteSession{
					CreatedBy: tt.claims.UUID, TTL: tt.body.TTL,
					ExpiresAt: expires, Email: tt.body.Email,
					Roles: roles,
				})
				is.NoErr(err)
				rdb.EXPECT().
					Set(ctx, tt.id, []byte(expectedSession), tt.body.Timeout).
					Return(redis.NewStatusResult("", nil))
			}
			err := invites.Create()(c)
			is.True(errors.Is(err, tt.expected))
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected == nil {
				var resp map[string]string
				is.NoErr(json.NewDecoder(rec.Body).Decode(&resp))
				is.True(len(resp["path"]) > 0)
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
				mockSessionGet(t, rdb, s).Times(1)
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
				mockSessionGet(t, rdb, s).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.AssignableToTypeOf(""))
			},
			expected: echo.ErrForbidden,
			internal: ErrInviteTTL,
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
				mockSessionGet(t, rdb, s).Times(1)
			},
		},
		{
			// Session not found
			expected: echo.ErrNotFound,
			internal: redis.Nil,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) {
				rdb.EXPECT().Get(
					gomock.AssignableToTypeOf(context.Background()),
					gomock.AssignableToTypeOf(""),
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
				Path:    &testPath{p: "invite"},
				RDB:     rdb,
				Encoder: base64.RawURLEncoding,
			}
			defer ctrl.Finish()

			id := "123"
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
			gomock.AssignableToTypeOf(""),
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
				mocks.rdb.EXPECT().Get(context.Background(), gomock.AssignableToTypeOf("")).Return(redis.NewStringResult("", redis.Nil))
			},
		},
		{
			name:     "expired ttl",
			session:  &inviteSession{TTL: 0},
			expected: echo.ErrForbidden,
			internal: ErrInviteTTL,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session).Times(1)
				mocks.rdb.EXPECT().Del(
					context.Background(),
					gomock.AssignableToTypeOf(""),
				).Return(redis.NewIntResult(0, nil))
			},
		},
		{
			name:     "Fail to update session with new ttl",
			session:  &inviteSession{TTL: 10},
			expected: echo.ErrInternalServerError,
			internal: redis.Nil,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", redis.Nil))
			},
		},
		{
			name:     "Fail to parse request body",
			session:  &inviteSession{TTL: 64},
			expected: echo.ErrBadRequest,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "no email",
			session:  &inviteSession{TTL: 12},
			login:    &Login{Password: "yeet", Username: "abc"},
			expected: ErrEmptyLogin,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "no password",
			session:  &inviteSession{TTL: -1},
			login:    &Login{Email: "yeet@yeet.com", Username: "abc"},
			expected: ErrEmptyLogin,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
			},
		},
		{
			name:     "email missmatch",
			session:  &inviteSession{TTL: 55, Email: "what@theheck.org"},
			login:    &Login{Email: "what@not_theheck.org", Password: "password1"},
			expected: ErrInviteEmailMissmatch,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
				mockSessionUpdate(t, mocks.rdb, tt.session).Return(redis.NewStatusResult("", nil))
			},
		},
		{
			name:     "failed to create user",
			session:  &inviteSession{TTL: -1},
			login:    &Login{Email: "a@a.it", Password: "123", Username: "test-user"},
			expected: echo.ErrInternalServerError, internal: randomError,
			mocks: func(t *testing.T, tt *table, mocks *mocks) {
				mockSessionGet(t, mocks.rdb, tt.session)
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
					mockSessionGet(t, mocks.rdb, tt.session),
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
						gomock.AssignableToTypeOf(&time.Time{}),
						gomock.AssignableToTypeOf(&time.Time{}),
					).Return(nil),
					mocks.rows.EXPECT().Close().Return(nil),
					mocks.rdb.EXPECT().Del(ctx, gomock.AssignableToTypeOf("")).Return(redis.NewIntResult(1, nil)),
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
				Path:    &testPath{p: "invite"},
				RDB:     rdb,
				Encoder: base64.RawURLEncoding,
			}
			defer ctrl.Finish()

			req := httptest.NewRequest("POST", "/", body(tt.login)).WithContext(ctx)
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
			is.Equal(rec.Code, http.StatusPermanentRedirect)
			is.Equal(rec.Header().Get("location"), "/")
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
					gomock.AssignableToTypeOf(""),
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
				mockSessionGet(t, rdb, tt.session).Times(1)
			},
		},
		{
			name:     "fail to delete session",
			id:       "123",
			claims:   &auth.Claims{UUID: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			session:  &inviteSession{CreatedBy: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			expected: echo.ErrInternalServerError,
			internal: redis.ErrClosed,
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				mockSessionGet(t, rdb, tt.session).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.AssignableToTypeOf("")).
					Return(redis.NewIntResult(0, redis.ErrClosed))
			},
		},
		{
			name:    "success",
			id:      "123",
			claims:  &auth.Claims{UUID: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			session: &inviteSession{CreatedBy: uuid.MustParse("c310417d-4d00-458d-927d-d4416d10c68f")},
			mocks: func(t *testing.T, rdb *mockredis.MockCmdable, tt *table) {
				mockSessionGet(t, rdb, tt.session).Times(1)
				rdb.EXPECT().Del(context.Background(), gomock.AssignableToTypeOf("")).Return(redis.NewIntResult(0, nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			// t.Skip()
			ctx := context.Background()
			is := is.New(t)
			ctrl := gomock.NewController(t)
			rdb := mockredis.NewMockCmdable(ctrl)
			invites := Invitations{
				Path:    &testPath{p: "invite"},
				RDB:     rdb,
				Encoder: base64.RawURLEncoding,
			}
			defer ctrl.Finish()

			req := httptest.NewRequest("DELETE", fmt.Sprintf("/invite/%s", tt.id), nil).WithContext(ctx)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.Set(auth.ClaimsContextKey, tt.claims)

			if tt.mocks != nil {
				tt.mocks(t, rdb, &tt)
			}
			err := invites.Delete()(c)
			is.True(errors.Is(tt.expected, err))
			is.Equal(tt.expected, err)
			if httpErr, ok := err.(*echo.HTTPError); ok && tt.internal != nil {
				is.True(errors.Is(httpErr.Internal, tt.internal))
			}
			if tt.expected != nil {
				return
			}
		})
	}
}

func TestInviteSession(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	rdb := mockredis.NewMockCmdable(ctrl)
	now := time.Now()
	invites := Invitations{
		RDB:     rdb,
		Encoder: base64.StdEncoding,
		Now:     func() time.Time { return now },
	}
	defer ctrl.Finish()

	uid := uuid.New()
	timeout := time.Second
	ttl := 3
	email := "t@t.com"

	k, err := invites.key()
	if err != nil {
		t.Fatal(err)
	}
	is.True(len(k) > 0)
	expires := now.Add(timeout).UnixMilli()
	rawSession, err := json.Marshal(&inviteSession{
		CreatedBy: uid, TTL: ttl,
		ExpiresAt: expires, Email: email,
	})
	is.NoErr(err)
	rdb.EXPECT().Set(ctx, k, rawSession, timeout).Return(redis.NewStatusResult("", nil))
	rdb.EXPECT().Get(ctx, k).Return(redis.NewStringResult(string(rawSession), nil))

	err = invites.set(ctx, k, timeout, &inviteSession{CreatedBy: uid, ExpiresAt: expires, TTL: ttl, Email: email})
	is.NoErr(err)
	session, err := invites.get(ctx, k)
	is.NoErr(err)
	is.Equal(session.CreatedBy[:], uid[:])
	is.Equal(session.TTL, ttl)
	is.Equal(session.Email, email)
	is.Equal(session.ExpiresAt, expires)
}

func mockSessionGet(t *testing.T, rdb *mockredis.MockCmdable, s *inviteSession) *gomock.Call {
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return rdb.EXPECT().Get(
		context.Background(),
		gomock.AssignableToTypeOf(""),
	).Return(redis.NewStringResult(string(raw), nil))
}
