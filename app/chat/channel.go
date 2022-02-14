package chat

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"nhooyr.io/websocket"
)

type Channel interface {
}

func NewChannel(store Store, rd redis.Cmdable, s *websocket.Conn) *channel {
	return &channel{store: store, rd: rd, s: s}
}

type channel struct {
	rd    redis.Cmdable
	s     *websocket.Conn
	store Store
}

func (c *channel) Listen(ctx context.Context) error {
	for {
		err := c.listen(ctx)
		if err != nil {
			return err
		}
	}
}

func (c *channel) listen(ctx context.Context) error {
	typ, r, err := c.s.Reader(ctx)
	if err != nil {
		logger.WithError(err).Warn("could not get reader")
		return err
	}
	fmt.Println(typ, r)
	return nil
}
