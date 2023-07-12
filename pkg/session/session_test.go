package session

import (
	"bytes"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"gopkg.hrry.dev/homelab/pkg/internal/mocks/mockredis"
)

type data struct {
	Name string
	ID   int
}

func init() { RegisterSerializable(&data{}) }

func Test(t *testing.T) {
	is := is.New(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	m := NewManager[data](
		"test-cookie",
		NewMemStore[data](time.Second),
		WithPath("/test-cookie-path"),
		WithSameSite(http.SameSiteStrictMode),
		WithDomain("example.com"),
		WithExpiration(time.Second),
		WithHTTPOnly(true),
		WithMaxAge(100),
		WithSecure(true),
	)
	ss := m.NewSession(&data{ID: 1, Name: "jimmy"})
	ctx := context.Background()
	err := ss.Save(ctx, rec)
	is.NoErr(err)
	req.Header.Set("Cookie", rec.Header().Get("Set-Cookie"))
	s1, err := m.Get(req)
	is.NoErr(err)
	is.Equal(ss.ID(), s1.ID())
	is.Equal(ss.Name(), s1.Name())
	cookie, err := req.Cookie(m.Name)
	is.NoErr(err)
	is.Equal(cookie.Name, m.Name)
	is.Equal(cookie.Value, ss.id)
	err = s1.Save(ctx, rec)
	is.NoErr(err)
	err = ss.Delete(ctx, rec)
	is.NoErr(err)
	err = m.Delete(rec, req)
	is.NoErr(err)
}

func TestNewSession(t *testing.T) {
	is := is.New(t)
	rec := httptest.NewRecorder()
	ctx := context.Background()
	m := NewManager[data](
		"test-cookie",
		NewMemStore[data](-1),
		WithPath("/cookie-path"),
		WithSameSite(http.SameSiteStrictMode),
	)
	ss := m.NewSession(&data{ID: 2, Name: "jerry"})
	err := ss.Save(ctx, rec)
	is.NoErr(err)
	is.True(ss.ID() != "")
	is.Equal(ss.Name(), "test-cookie")
	is.Equal(ss.Value.ID, 2)
	is.Equal(ss.Value.Name, "jerry")
}

func TestManager_Get(t *testing.T) {
	is := is.New(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	m := NewManager[data]("test-cookie", NewMemStore[data](time.Microsecond))

	err := m.SetValue(rec, req, &data{ID: 3, Name: "johnny"})
	is.NoErr(err)
	_, err = m.Get(req)
	is.Equal(err, http.ErrNoCookie)

	res := rec.Result()
	defer res.Body.Close()
	cookie := res.Cookies()[0]
	req.AddCookie(cookie)

	s, err := m.Get(req)
	is.NoErr(err)
	is.Equal(s.name, m.Name)
	is.Equal(s.name, cookie.Name)
	is.Equal(s.id, cookie.Value)
	is.Equal(s.Value.ID, 3)
	is.Equal(s.Value.Name, "johnny")
	s.Set(&data{ID: 69, Name: "nice"})
	is.Equal(s.Value.ID, 69)
	is.Equal(s.Value.Name, "nice")
}

func TestManager_Get_timeout(t *testing.T) {
	defer func() {
		now = time.Now
		tidyTime = time.Second
	}()
	tidyTime = time.Millisecond
	now = func() time.Time {
		now := time.Now()
		return time.Unix(now.Unix()+1000, 0)
	}
	is := is.New(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	m := NewManager[data]("test-cookie", NewMemStore[data](time.Microsecond))

	err := m.SetValue(rec, req, &data{ID: 3, Name: "johnny"})
	is.NoErr(err)

	res := rec.Result()
	defer res.Body.Close()
	cookie := res.Cookies()[0]
	req.AddCookie(cookie)

	time.Sleep(time.Millisecond * 5)
	s, err := m.Get(req)
	is.Equal(err, ErrSessionNotFound)
	is.Equal(s, nil)
}

func TestManager_Delete(t *testing.T) {
	is := is.New(t)
	ctx := context.Background()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	m := NewManager[data](
		"test-cookie",
		NewMemStore[data](time.Second),
		WithPath("/test-cookie-path"),
		WithSameSite(http.SameSiteStrictMode),
		WithDomain("example.com"),
		WithExpiration(time.Second),
		WithHTTPOnly(true),
		WithMaxAge(100),
		WithSecure(true),
	)

	err := m.SetValue(rec, req, &data{ID: 3, Name: "johnny"})
	is.NoErr(err)
	res := rec.Result()
	defer res.Body.Close()
	cookie := res.Cookies()[0]
	v, err := m.Store.Get(ctx, m.key(cookie.Value))
	is.NoErr(err)
	is.Equal(v.ID, 3)
	is.Equal(v.Name, "johnny")

	req.AddCookie(cookie)
	rec.Flush()
	err = m.Delete(rec, req)
	is.NoErr(err)
	_, err = m.Store.Get(ctx, cookie.Value)
	is.Equal(err, ErrSessionNotFound)
}

func TestManager_UpdateValue(t *testing.T) {
	is := is.New(t)
	// ctx := context.Background()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	m := NewManager[data](
		"test-cookie",
		NewMemStore[data](time.Second),
		WithPath("/test-cookie-path"),
		WithSameSite(http.SameSiteStrictMode),
		WithDomain("example.com"),
		WithExpiration(time.Second),
		WithHTTPOnly(true),
		WithMaxAge(100),
		WithSecure(true),
	)
	err := m.SetValue(rec, req, &data{ID: 69, Name: "nice"})
	is.NoErr(err)

	res := rec.Result()
	defer res.Body.Close()
	req.AddCookie(res.Cookies()[0])
	v, err := m.GetValue(req)
	is.NoErr(err)
	is.Equal(v.ID, 69)
	is.Equal(v.Name, "nice")

	err = m.UpdateValue(rec, req, &data{ID: 6969, Name: "not nice"})
	is.NoErr(err)
	v, err = m.GetValue(req)
	is.NoErr(err)
	is.Equal(v.ID, 6969)
	is.Equal(v.Name, "not nice")
}

type cmder[T any] interface {
	redis.Cmder
	SetVal(T)
}

func initCmd[T any](cmd cmder[T], val T, err error) {
	cmd.SetVal(val)
	cmd.SetErr(err)
}

func statusCmd(ctx context.Context, err error) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	initCmd[string](cmd, "", err)
	return cmd
}

func intCmd(ctx context.Context, err error) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx)
	initCmd[int64](cmd, 0, err)
	return cmd
}

func strCmd(ctx context.Context, val string, err error) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	initCmd[string](cmd, val, err)
	return cmd
}

func gobit[T any](v *T) string {
	var b bytes.Buffer
	err := gob.NewEncoder(&b).Encode(v)
	if err != nil {
		panic(err)
	}
	return b.String()
}

func TestRedisStore(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rd := mockredis.NewMockUniversalClient(ctrl)
	rs := NewRedisStore[data](rd, time.Second)
	ctx := context.Background()

	t.Run(fmt.Sprintf("%s_Set", t.Name()), func(t *testing.T) {
		is := is.New(t)
		in := &data{ID: 1, Name: "one"}
		rd.EXPECT().
			Set(ctx, "one", gobit(in), time.Second).
			Return(statusCmd(ctx, nil))
		err := rs.Set(ctx, "one", in)
		is.NoErr(err)

		in = &data{ID: 2, Name: "two"}
		rd.EXPECT().
			Set(ctx, "two", gobit(in), time.Second).
			Return(statusCmd(ctx, redis.Nil))
		err = rs.Set(ctx, "two", in)
		is.Equal(err, redis.Nil)
	})

	t.Run(fmt.Sprintf("%s_Get", t.Name()), func(t *testing.T) {
		is := is.New(t)
		in := &data{ID: 1, Name: "one"}
		rd.EXPECT().
			Get(ctx, "one").
			Return(strCmd(ctx, gobit(in), nil))
		v, err := rs.Get(ctx, "one")
		is.NoErr(err)
		is.Equal(v.ID, in.ID)
		is.Equal(v.Name, in.Name)

		rd.EXPECT().
			Get(ctx, "two").
			Return(strCmd(ctx, "", redis.Nil))
		v, err = rs.Get(ctx, "two")
		is.Equal(err, ErrSessionNotFound)
		is.True(v == nil)

		demoErr := errors.New("demo error")
		rd.EXPECT().
			Get(ctx, "three").
			Return(strCmd(ctx, "", demoErr))
		v, err = rs.Get(ctx, "three")
		is.Equal(err, demoErr)
		is.True(v == nil)
	})

	t.Run(fmt.Sprintf("%s_Del", t.Name()), func(t *testing.T) {
		is := is.New(t)
		rd.EXPECT().
			Del(ctx, "one").
			Return(intCmd(ctx, nil))
		err := rs.Del(ctx, "one")
		is.NoErr(err)

		rd.EXPECT().
			Del(ctx, "two").
			Return(intCmd(ctx, redis.Nil))
		err = rs.Del(ctx, "two")
		is.Equal(err, ErrSessionNotFound)

		demoErr := errors.New("demo error")
		rd.EXPECT().
			Del(ctx, "three").
			Return(intCmd(ctx, demoErr))
		err = rs.Del(ctx, "three")
		is.Equal(err, demoErr)
	})
}
