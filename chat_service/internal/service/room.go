package service

import (
	"context"
	"log/slog"
	"sync"

	"chat_service/internal/domain"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"
)

type Room interface {
	AddUser(user User) error
	RemoveUser(user User) error
	IsEmpty() bool
	Broadcast(ctx context.Context, msg *domain.Message)
	Run(ctx context.Context)
	ID() string
	Stop()
	GetUsernames() []string
}

type room struct {
	id        string
	users     map[string]User /// TODO need or true/false enough to close room
	stopCh    chan struct{}
	in        chan *domain.Message
	msgRepo   repository.MessageRepository
	logger    *slog.Logger
	mu        sync.RWMutex
	closeOnce sync.Once
	ps        pubsub.PubSub
}

func NewRoom(
	id string,
	msgRepo repository.MessageRepository,
	ps pubsub.PubSub,
	logger *slog.Logger,
) Room {
	return &room{
		id:      id,
		users:   make(map[string]User),
		stopCh:  make(chan struct{}),
		msgRepo: msgRepo,
		logger:  logger,
		in:      make(chan *domain.Message),
		ps:      ps,
	}
}

func (r *room) AddUser(user User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.users[user.ID()]
	if ok {
		return domain.ErrUserExists
	}
	r.users[user.ID()] = user

	return nil
}

func (r *room) RemoveUser(user User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	_, ok := r.users[user.ID()]
	if !ok {
		return domain.ErrUserNotFound
	}
	delete(r.users, user.ID())

	return nil
}

func (r *room) IsEmpty() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.users) == 0
}

func (r *room) Broadcast(ctx context.Context, msg *domain.Message) {
	select {
	case r.in <- msg:
	case <-r.stopCh:
	case <-ctx.Done():
	}
}

func (r *room) Run(ctx context.Context) {
	const op = "chat.service.room.run"
	logger := r.logger.With(slog.String("op", op))

	msgCh, unsub := r.ps.Subscribe(ctx, r.channelID())
	defer unsub()

	for {
		select {
		case msg := <-r.in:
			go r.msgRepo.WriteMessage(ctx, msg)
			r.ps.Publish(ctx, r.channelID(), msg)

		case msg, ok := <-msgCh:
			if !ok {
				continue
			}
			r.sendAll(ctx, msg)

		case <-ctx.Done():
			logger.Info("ctx done case", "roomID", r.id)
			return
		case <-r.stopCh:
			logger.Info("stop command", "roomID", r.id)
			return
		}
	}
}

func (r *room) sendAll(ctx context.Context, msg *domain.Message) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, user := range r.users {
		user.Write(ctx, msg)
	}
}

func (r *room) ID() string {
	return r.id
}

func (r *room) Stop() {
	r.mu.Lock()
	r.closeOnce.Do(func() { close(r.stopCh) })
	r.mu.Unlock()
}

// TODO used only in tests
func (r *room) GetUsernames() []string {
	usernames := make([]string, 0, len(r.users))
	r.mu.RLock()
	for _, user := range r.users {
		usernames = append(usernames, user.Name())
	}
	r.mu.RUnlock()
	return usernames
}

func (r *room) channelID() string {
	return "room:" + r.id
}
