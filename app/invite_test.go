package app

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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
	"github.com/matryer/is"
	"harrybrown.com/internal/mocks/mockredis"
)

type testPath struct{ p string }

func (tp *testPath) Path(id string) string {
	return filepath.Join("/", tp.p, id)
}

func (tp *testPath) GetID(req *http.Request) string {
	list := strings.Split(req.URL.Path, string(filepath.Separator))
	return list[2]
}

func TestInviteCreate(t *testing.T) {

}

func TestInviteAccept(t *testing.T) {
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

	e := echo.New()
	req := httptest.NewRequest("GET", invites.Path.Path("123"), nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handler := invites.Accept([]byte(`{"email":"{{ .Email }}","expires":{{ .ExpiresAt }}}`), "text/html")

	rdb.EXPECT().Get(ctx, "123").Return(redis.NewStringResult(`{"tl":5,"to":10,"e":"t@t.io"}`, nil))
	err := handler(c)
	is.NoErr(err)
	data := struct {
		Email   string `json:"email"`
		Expires int64  `json:"expires"`
	}{}
	is.NoErr(json.NewDecoder(rec.Body).Decode(&data))
	is.Equal(data.Email, "t@t.io")
}

func TestInviteSignUp(t *testing.T) {

}

func TestInviteSession(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	rdb := mockredis.NewMockCmdable(ctrl)
	invites := Invitations{
		Path:    &testPath{p: "invite"},
		RDB:     rdb,
		Encoder: base64.StdEncoding,
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
	expires := time.Now().Add(timeout).UnixMilli()
	rawSession := fmt.Sprintf(`{"cb":"%s","tl":%d,"to":%d,"ex":%d,"e":"%s"}`, uid, ttl, timeout, expires, email)
	rdb.EXPECT().Set(ctx, k, []byte(rawSession), timeout).Return(redis.NewStatusResult("", nil))
	rdb.EXPECT().Get(ctx, k).Return(redis.NewStringResult(rawSession, nil))

	err = invites.put(ctx, k, &inviteSession{CreatedBy: uid, Timeout: timeout, ExpiresAt: expires, TTL: ttl, Email: email})
	is.NoErr(err)
	session, err := invites.get(ctx, k)
	is.NoErr(err)
	is.Equal(session.CreatedBy[:], uid[:])
	is.Equal(session.Timeout, timeout)
	is.Equal(session.TTL, ttl)
	is.Equal(session.Email, email)
	is.Equal(session.ExpiresAt, expires)
}
