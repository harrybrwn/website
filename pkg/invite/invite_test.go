package invite

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"gopkg.hrry.dev/homelab/pkg/auth"
	"gopkg.hrry.dev/homelab/pkg/internal/mocks/mockredis"
	"gopkg.hrry.dev/homelab/pkg/internal/mockutil"
)

func init() {
	logger.SetOutput(io.Discard)
}

func TestInviteSessionStore_Get(t *testing.T) {
	// logger.SetOutput(io.Discard)
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rdb := mockredis.NewMockCmdable(ctrl)
	s := SessionStore{RDB: rdb, Prefix: "inv", KeyGen: DefaultKeyGen}
	ctx := context.Background()

	// Decrement TTL
	expectedSession := Session{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 10}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	updated := Session{CreatedBy: expectedSession.CreatedBy, Roles: expectedSession.Roles, TTL: 9}
	mockSessionSet(t, rdb, gomock.Eq("inv:123"), gomock.Eq(time.Duration(redis.KeepTTL)), &updated).Return(redis.NewStatusResult("", nil))
	session, err := s.Get(ctx, "123")
	is.NoErr(err)
	is.Equal(*session, updated)
	is.Equal(session.TTL, expectedSession.TTL-1)

	// Delete because of expired TTL
	expectedSession = Session{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 0}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	rdb.EXPECT().Del(ctx, "inv:123").Return(redis.NewIntResult(0, nil))
	session, err = s.Get(ctx, "123")
	is.Equal(err, ErrInviteTTL)
	is.Equal(session, nil)

	// Delete because of expired TTL: failed delete
	expectedSession = Session{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: 0}
	mockSessionGet(t, rdb, gomock.Eq("inv:123"), &expectedSession)
	rdb.EXPECT().Del(ctx, "inv:123").Return(redis.NewIntResult(0, errors.New("this is some random error")))
	session, err = s.Get(ctx, "123")
	is.Equal(err, ErrInviteTTL)
	is.Equal(session, nil)

	// Ignore negative TTL
	expectedSession = Session{CreatedBy: uuid.New(), Roles: []auth.Role{auth.RoleFamily}, TTL: -1}
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
	var now = time.Now
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
					&Session{TTL: defaultInviteTTL, ExpiresAt: now().Add(defaultInviteTimeout).UnixMilli(), CreatedBy: tt.uuid},
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
					&Session{TTL: defaultInviteTTL, ExpiresAt: now().Add(defaultInviteTimeout).UnixMilli(), CreatedBy: tt.uuid},
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
					&Session{TTL: 53, ExpiresAt: now().Add(time.Hour * 9).UnixMilli(), CreatedBy: tt.uuid},
				).Return(redis.NewStatusResult("", nil))
			},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d_%s", t.Name(), i, tt.name), func(t *testing.T) {
			is := is.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			rd := mockredis.NewMockCmdable(ctrl)
			store := SessionStore{Prefix: "iss_create", RDB: rd, KeyGen: DefaultKeyGen}
			ctx := context.Background()
			tt.mock(t, &tt, rd)
			store.Now = now
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
	store := SessionStore{RDB: rd, Prefix: "i", KeyGen: DefaultKeyGen}

	uid := uuid.New()

	mockSessionGet(t, rd, gomock.Eq("i:abc"), &Session{CreatedBy: uid})
	rd.EXPECT().Del(ctx, "i:abc").Return(redis.NewIntResult(0, nil))
	err := store.OwnerDel(ctx, "abc", uid)
	is.NoErr(err)

	mockSessionGet(t, rd, gomock.Eq("i:abc"), &Session{CreatedBy: uid})
	err = store.OwnerDel(ctx, "abc", uuid.New())
	is.True(errors.Is(err, ErrSessionOwnership))
}

func TestInviteSessionStore_List(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	var err error
	is := is.New(t)
	rdb := mockredis.NewMockCmdable(ctrl)
	s := SessionStore{RDB: rdb, Prefix: "inv", KeyGen: DefaultKeyGen}
	ctx := context.Background()
	keys := []string{"inv:1", "inv:2", "inv:3"}
	expected := []*Session{
		{ID: "1", CreatedBy: uuid.New(), ExpiresAt: 1000, Roles: []auth.Role{auth.RoleAdmin}},
		{ID: "2", CreatedBy: uuid.New(), ExpiresAt: 1001, Roles: []auth.Role{auth.RoleDefault}},
		{ID: "3", CreatedBy: uuid.New(), ExpiresAt: 1002, Roles: []auth.Role{auth.RoleFamily}},
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

func mockSessionGet(t *testing.T, rdb *mockredis.MockCmdable, key gomock.Matcher, s *Session) *gomock.Call {
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
	s *Session,
) *gomock.Call {
	t.Helper()
	raw, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	return rd.EXPECT().Set(context.Background(), key, raw, timeout)
}
