package db

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"gopkg.hrry.dev/homelab/pkg/internal/mocks/mockrows"
)

func TestScanOne(t *testing.T) {
	var errTestError = errors.New("test error")
	run := func(name string, fn func(t *testing.T, r *mockrows.MockRows)) {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			r := mockrows.NewMockRows(ctrl)
			fn(t, r)
		})
	}

	run("happy path", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(true)
		r.EXPECT().Scan().Return(nil)
		r.EXPECT().Close().Return(errTestError)
		err := ScanOne(r)
		is.True(errors.Is(err, errTestError))
	})

	run("scan error", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(true)
		r.EXPECT().Scan().Return(errTestError)
		r.EXPECT().Close().Return(nil)
		err := ScanOne(r)
		is.True(errors.Is(err, errTestError))
	})

	run("scan error close error", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(true)
		r.EXPECT().Scan().Return(errTestError)
		r.EXPECT().Close().Return(errTestError)
		err := ScanOne(r)
		is.True(errors.Is(err, errTestError))
	})

	run("no next no rows", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(false)
		r.EXPECT().Err().Return(nil)
		r.EXPECT().Close().Return(nil)
		err := ScanOne(r)
		is.True(errors.Is(err, sql.ErrNoRows))
	})

	run("no next no rows error", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(false)
		r.EXPECT().Err().Return(nil)
		r.EXPECT().Close().Return(errTestError)
		err := ScanOne(r)
		is.True(errors.Is(err, sql.ErrNoRows))
	})

	run("no next with Err", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(false)
		r.EXPECT().Err().Return(errTestError)
		r.EXPECT().Close().Return(nil)
		err := ScanOne(r)
		is.True(errors.Is(err, errTestError))
	})

	run("no next with both Err", func(t *testing.T, r *mockrows.MockRows) {
		is := is.New(t)
		r.EXPECT().Next().Return(false)
		r.EXPECT().Err().Return(errTestError)
		r.EXPECT().Close().Return(errTestError)
		err := ScanOne(r)
		is.True(errors.Is(err, errTestError))
	})
}
