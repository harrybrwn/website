package chat

import (
	"github.com/go-redis/redis/v8"
)

type Channel interface{}

func NewChannel(rd redis.UniversalClient, room, user int) *channel {
	return &channel{RDB: rd, Room: room, User: user}
}

type channel struct {
	RDB  redis.UniversalClient
	Room int
	User int
}

// Pub publishes a message to the channel
func (c *channel) Pub(msg *Message) error {
	return nil
}

// Sub subscribes to a channel to listen to new m
func (c *channel) Sub() (<-chan Message, error) {
	return nil, nil
}
