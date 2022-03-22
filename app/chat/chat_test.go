package chat

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang/mock/gomock"
	"github.com/matryer/is"
	"harrybrown.com/internal/mocks/mockredis"
)

func Test(t *testing.T) {
	t.Skip()
	rd := redis.NewClient(&redis.Options{
		Password: "configure-the-vampire-clones",
		Addr:     "localhost:6379",
	})
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	pubsub := rd.Subscribe(ctx)              // create a new pubsub
	err := pubsub.PSubscribe(ctx, "*:msg:*") // subscribe to "*" channel
	if err != nil {
		t.Fatal(err)
	}
	ch := pubsub.Channel()
	for msg := range ch {
		fmt.Printf("channel:%s pat:%s payload:%s\n", msg.Channel, msg.Pattern, msg.Payload)
	}
}

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
