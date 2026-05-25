package pubsub

import (
	"context"

	"chat_service/internal/domain"
)

type PubSub interface {
	Publish(ctx context.Context, channel string, msg *domain.Message) error
	Subscribe(ctx context.Context, channel string) (chan *domain.Message, func())
}
