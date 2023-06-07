package app

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"harrybrown.com/pkg/internal/mocks/mockdb"
	"harrybrown.com/pkg/internal/mocks/mockredis"
)

func init() {
	logger.SetLevel(logrus.ErrorLevel)
}

func TestHits(t *testing.T) {
	var (
		is   = is.New(t)
		ctrl = gomock.NewController(t)
		db   = mockdb.NewMockDB(ctrl)
		rows = mockdb.NewMockRows(ctrl)
		rd   = mockredis.NewMockCmdable(ctrl)
		e    = echo.New()
		ctx  = context.Background()
		hc   = &hitsCache{rd: rd, timeout: time.Minute}
	)
	defer ctrl.Finish()
	defer silent()()
	type table struct {
		u string
	}
	for _, tab := range []table{
		{u: "/api/test"},
		{u: "/"},
		{u: ""},
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("GET", "/api/hits", nil).WithContext(ctx)
		c := e.NewContext(req, rec)
		c.QueryParams().Set("u", tab.u)
		exp := tab.u
		if exp == "" {
			exp = "/"
		}
		key := fmt.Sprintf("hits:%s", exp) // cache key

		rd.EXPECT().Incr(ctx, key).Return(redis.NewIntResult(0, errors.New("asdf")))
		db.EXPECT().QueryContext(ctx, hitsQuery, exp).Return(rows, nil)
		rows.EXPECT().Next().Return(true)
		rows.EXPECT().Scan(gomock.Any()).Return(nil)
		rows.EXPECT().Close().Return(nil).Times(1)
		rd.EXPECT().Set(ctx, key, int64(0), hc.timeout).Return(redis.NewStatusResult("", nil))

		is.NoErr(Hits(db, hc, logger)(c))
		is.Equal(rec.Code, 200)
		is.True(strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))
		is.Equal("{\"count\":0}\n", rec.Body.String())

		rec = httptest.NewRecorder()
		req = httptest.NewRequest("GET", "/api/hits", nil).WithContext(ctx)
		c = e.NewContext(req, rec)
		c.QueryParams().Set("u", tab.u)
		rd.EXPECT().Incr(ctx, key).Return(redis.NewIntResult(int64(2), nil))
		is.NoErr(Hits(db, hc, logger)(c))
		is.Equal(rec.Code, 200)
		is.True(strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"))
		is.Equal("{\"count\":2}\n", rec.Body.String())
	}
}

func TestHitsCache(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	rd := mockredis.NewMockCmdable(ctrl)
	cache := &hitsCache{rd: rd, timeout: time.Second}
	ctx := context.Background()
	key := "one"
	rd.EXPECT().Incr(ctx, key).Return(redis.NewIntResult(1, nil))
	n, err := cache.Next(ctx, key)
	is.True(err != nil) // incr resulting in 1 means not found, should be error
	is.Equal(n, int64(0))
	rd.EXPECT().Incr(ctx, key).Return(redis.NewIntResult(5, nil))
	n, err = cache.Next(ctx, key)
	is.NoErr(err)
	is.Equal(n, int64(5))
}

var (
	intptr      *int
	strptr      *string
	bytesptr    *[]byte
	durationPtr *time.Duration
)

func TestLogList(t *testing.T) {
	var (
		ctx = context.Background()
	)
	type table struct {
		errs  []error
		query url.Values
		prep  func(db *mockdb.MockDB, rows *mockdb.MockRows)
	}
	expectScan := func(rows *mockdb.MockRows) *gomock.Call {
		return rows.EXPECT().Scan(
			gomock.AssignableToTypeOf(intptr),
			gomock.AssignableToTypeOf(strptr),
			gomock.AssignableToTypeOf(intptr),
			gomock.AssignableToTypeOf(strptr),
			gomock.AssignableToTypeOf(strptr),
			gomock.AssignableToTypeOf(&sql.NullString{}),
			gomock.AssignableToTypeOf(strptr),
			gomock.AssignableToTypeOf(durationPtr),
			gomock.Any(),
			gomock.AssignableToTypeOf(&time.Time{}),
			gomock.AssignableToTypeOf(&uuid.UUID{}),
		)
	}

	for i, tt := range []table{
		{
			errs:  []error{},
			query: url.Values{"limit": {"12"}, "offset": {"0"}},
			prep: func(db *mockdb.MockDB, rows *mockdb.MockRows) {
				db.EXPECT().QueryContext(ctx, getLogsQuery+" LIMIT $2", []interface{}{0, 12}).Return(rows, nil)
				rows.EXPECT().Next().Times(1).Return(true)
				expectScan(rows).Do(func(v ...interface{}) {}).Return(nil)
				rows.EXPECT().Next().Times(1).Return(false)
				rows.EXPECT().Close().Return(nil)
			},
		},
	} {
		if tt.prep == nil {
			tt.prep = func(*mockdb.MockDB, *mockdb.MockRows) {}
		}
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			db := mockdb.NewMockDB(ctrl)
			rows := mockdb.NewMockRows(ctrl)
			e := echo.New()
			rec := httptest.NewRecorder()
			req := httptest.NewRequest("GET", "/", nil).WithContext(ctx)
			req.URL.RawQuery = tt.query.Encode()
			handler := LogListHandler(db)
			c := e.NewContext(req, rec)
			tt.prep(db, rows)
			err := handler(c)
			checkErrs(t, tt.errs, err)
			if len(tt.errs) > 0 {
				return
			}
		})
	}
}

func checkErrs(t *testing.T, expected []error, err error) (stop bool) {
	t.Helper()
	if len(expected) > 0 {
		for _, er := range expected {
			if !errors.Is(err, er) {
				t.Errorf("expected \"%v\", got \"%v\"", er, err)
			}
		}
		return true
	} else {
		if err != nil {
			t.Error(err)
		}
	}
	return false
}

func silent() func() {
	out := logger.Out
	logger.SetOutput(io.Discard)
	return func() { logger.SetOutput(out) }
}

func body(i interface{}) io.Reader {
	var b bytes.Buffer
	if err := json.NewEncoder(&b).Encode(i); err != nil {
		panic(err)
	}
	return &b
}
