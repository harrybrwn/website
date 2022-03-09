package chat

import (
	"github.com/go-redis/redis/v8"
)

type Channel interface {
	Pub(*Message) error
	Sub() (<-chan Message, error)
}

func NewChannel(rd redis.Cmdable, id int) *channel {
	return &channel{rd: rd, id: id}
}

type channel struct {
	rd redis.Cmdable
	// id is the room id
	id int
}

// Pub publishes a message to the channel
func (c *channel) Pub(msg *Message) error {
	return nil
}

// Sub subscribes to a channel to listen to new m
func (c *channel) Sub() (<-chan Message, error) {
	return nil, nil
}
