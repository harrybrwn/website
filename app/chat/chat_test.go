package chat

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-redis/redis/v8"
)

func Test(t *testing.T) {
	rd := redis.NewClient(&redis.Options{
		Password: "configure-the-vampire-clones",
		Addr:     "localhost:6379",
	})
	ctx := context.Background()
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
