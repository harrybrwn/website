package chat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"github.com/pkg/errors"
	"harrybrown.com/internal/mocks/mockredis"
	"harrybrown.com/internal/mocks/mockws"
	"harrybrown.com/pkg/ws"
)

func TestPubSub_Sub(t *testing.T) {
	type table struct {
	}
	for i, tt := range []*table{
		{},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			var (
				is   = is.New(t)
				ctrl = gomock.NewController(t)
				rdb  = mockredis.NewMockUniversalClient(ctrl)
				ps   = pubsub{RDB: rdb, Room: 1, User: 2}
			)
			defer ctrl.Finish()
			is.True(ps.RDB != nil && tt != nil)
		})
	}
}

func TestPubSub_Pub(t *testing.T) {
	type table struct {
		room     int
		user     int
		msg      *Message
		expected error
	}
	for i, tt := range []*table{
		{
			room: 1, user: 2,
			msg: &Message{Room: 1, UserID: 2, ID: 1, Body: "Hello bitch", CreatedAt: time.Now()},
		},
		{
			room: 4, user: 20,
			msg: &Message{Room: 4, UserID: 20, ID: 2, Body: "Hello?", CreatedAt: time.Now()},
		},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			var (
				is   = is.New(t)
				ctrl = gomock.NewController(t)
				rdb  = mockredis.NewMockUniversalClient(ctrl)
				ps   = pubsub{RDB: rdb, Room: tt.room, User: tt.user}
				ctx  = context.Background()
			)
			defer ctrl.Finish()
			is.True(ps.RDB != nil && tt != nil)
			raw, err := json.Marshal(tt.msg)
			is.True(errors.Is(err, tt.expected))
			if tt.expected != nil {
				return
			}
			rdb.EXPECT().
				Publish(ctx, fmt.Sprintf("room:%d:user:%d", tt.room, tt.user), raw).
				Return(redis.NewIntResult(0, nil))
			err = ps.Pub(ctx, tt.msg)
			is.NoErr(err)
		})
	}
}

func TestSocket_Send(t *testing.T) {
	type table struct {
		msg        *Message
		writeError error
		closeError error
		expected   error
	}
	testErr := errors.New("this is a test error")
	for i, tt := range []table{
		{msg: &Message{ID: 1, Body: "heebeejeebees"}},
		{msg: &Message{ID: 2, Body: "what?2"}, expected: nil},
		{msg: &Message{ID: 3, Body: "what?3"}, writeError: testErr, expected: testErr},
		{msg: &Message{ID: 4, Body: "what?4"}, closeError: testErr, expected: testErr},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := mockws.NewMockConnection(ctrl)
			s := NewSocket(conn)
			ctx := context.Background()

			var buf = writeCloser{err: tt.closeError}
			conn.EXPECT().
				Writer(ctx, ws.MessageText).
				Return(&buf, tt.writeError)
			err := s.Send(ctx, tt.msg)
			is.True(errors.Is(err, tt.expected))
			if tt.expected != nil {
				return
			}
			is.True(buf.Len() > 0)
			var msg Message
			b := buf.Bytes()
			is.Equal(b, msgBuf(tt.msg).Bytes())
			is.NoErr(json.Unmarshal(b, &msg))
			is.Equal(msg.ID, tt.msg.ID)
			is.Equal(msg.Room, tt.msg.Room)
			is.Equal(msg.UserID, tt.msg.UserID)
			is.Equal(msg.Body, tt.msg.Body)
		})
	}
}

func TestSocket_Recv(t *testing.T) {
	type table struct {
		msg       Message
		expected  error
		readError error
	}
	for i, tt := range []table{
		{msg: Message{ID: 1, Body: "what?"}, expected: nil},
		{msg: Message{}, expected: ErrEmptyBody},
		{msg: Message{Body: "test"}, expected: ErrEmptyBody, readError: ErrEmptyBody},
	} {
		t.Run(fmt.Sprintf("%s_%d", t.Name(), i), func(t *testing.T) {
			is := is.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			conn := mockws.NewMockConnection(ctrl)
			s := NewSocket(conn)
			ctx := context.Background()
			conn.EXPECT().
				Reader(ctx).
				Return(ws.MessageText, msgBuf(&tt.msg), tt.readError)
			msg, err := s.Recv(ctx)
			is.True(errors.Is(err, tt.expected))
			if tt.expected != nil {
				return
			}
			is.True(msg != nil)
			is.True(!msg.CreatedAt.IsZero())
			is.Equal(msg.ID, tt.msg.ID)
			is.Equal(msg.Body, tt.msg.Body)
			is.Equal(msg.Room, tt.msg.Room)
			is.Equal(msg.UserID, tt.msg.UserID)
		})
	}
}

func msgBuf(msg *Message) *bytes.Buffer {
	var b bytes.Buffer
	err := json.NewEncoder(&b).Encode(msg)
	if err != nil {
		panic(err)
	}
	return &b
}

type writeCloser struct {
	bytes.Buffer
	err error
}

func (wc *writeCloser) Close() error { return wc.err }
