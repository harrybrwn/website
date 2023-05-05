package app

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"harrybrown.com/pkg/internal/mocks/mockdb"
)

func Test(t *testing.T) {
}

func TestInsertLogs(t *testing.T) {
	is := is.New(t)
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	db := mockdb.NewMockDB(ctrl)
	ctx := context.Background()
	logManager := LogManager{db: db}

	l := RequestLog{
		Status:  http.StatusTeapot,
		IP:      string(net.IPv4(10, 0, 0, 69)),
		Referer: "/some/path",
		Error:   errors.New("test error"),
		UserID:  uuid.New(),
	}
	db.EXPECT().ExecContext(
		ctx, insertLogQuery,
		l.Method, l.Status, l.IP, l.URI, l.Referer,
		l.UserAgent, l.Latency, l.Error.Error(), l.UserID,
	).Return(nil, nil)
	err := logManager.Write(ctx, &l)
	is.NoErr(err)

	expectedErr := errors.New("a testing error")
	db.EXPECT().ExecContext(
		ctx, insertLogQuery, l.Method, l.Status, l.IP, l.URI, l.Referer,
		l.UserAgent, l.Latency, l.Error.Error(), l.UserID,
	).Return(nil, expectedErr)
	err = logManager.Write(ctx, &l)
	is.Equal(expectedErr, err)
}
