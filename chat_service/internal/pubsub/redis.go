package pubsub

import (
	"context"
	"encoding/json"
	"log"

	"github.com/redis/go-redis/v9"

	"chat_service/internal/domain"
)

type pubsub struct {
	rdb redis.UniversalClient
}

func NewPubSub(rdb redis.UniversalClient) PubSub {
	return &pubsub{rdb: rdb}
}

func (ps *pubsub) Subscribe(ctx context.Context, channel string) (messages chan *domain.Message, unsubscribe func()) {
	pubsub := ps.rdb.Subscribe(ctx, channel)
	out := make(chan *domain.Message, 64)
	go func() {
		defer close(out)
		redisCh := pubsub.Channel()

		for {
			select {
			case m, ok := <-redisCh:
				if !ok {
					return
				}
				var msg domain.Message
				if err := json.Unmarshal([]byte(m.Payload), &msg); err != nil {
					continue
				}

				select {
				case out <- &msg:
				default:
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, func() {
		if err := pubsub.Close(); err != nil {
			log.Printf("error closing conn: %v", err)
		}
	}
}

func (ps *pubsub) Publish(ctx context.Context, channel string, msg *domain.Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return ps.rdb.Publish(ctx, channel, data).Err()
}
