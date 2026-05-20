package pubsub

import (
	"chat_service/internal/domain"
	"context"
)

type PubSub interface {
	Publish(ctx context.Context, channel string, msg *domain.Message) error
	Subscribe(ctx context.Context, channel string) (chan *domain.Message, func())
}
