package service

import (
	"chat_service/internal/domain"
	"chat_service/internal/pubsub"
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func startRedis(t *testing.T) *redis.Client {
	t.Helper()

	ctr, err := tcredis.Run(t.Context(), "redis:7-alpine")
	require.NoError(t, err)
	t.Cleanup(func() { ctr.Terminate(t.Context()) })

	dsn, err := ctr.ConnectionString(t.Context())
	require.NoError(t, err)

	opt, err := redis.ParseURL(dsn)
	require.NoError(t, err)

	return redis.NewClient(opt)
}

type nopMsgRepo struct{}

func (n *nopMsgRepo) WriteMessage(_ context.Context, _ *domain.Message) error { return nil }
func (n *nopMsgRepo) GetMessagesBefore(_ context.Context, _ string, _ time.Time) ([]*domain.Message, error) {
	return nil, nil
}

type captureUser struct {
	id       string
	received chan *domain.Message
}

func (u *captureUser) ID() string                                  { return u.id }
func (u *captureUser) Name() string                                { return u.id }
func (u *captureUser) AddRoom(Room) error                          { return nil }
func (u *captureUser) DeleteRoom(Room) error                       { return nil }
func (u *captureUser) Rooms() map[string]Room                      { return nil }
func (u *captureUser) Listen(_ context.Context, _ *sync.WaitGroup) {}
func (u *captureUser) Stop()                                       {}
func (u *captureUser) Write(_ context.Context, msg *domain.Message) error {
	u.received <- msg
	return nil
}

func Test_CrossInstance_MessageDelivery(t *testing.T) {
	rdb := startRedis(t)
	ps := pubsub.NewPubSub(rdb)

	alice := &captureUser{id: "alice-id", received: make(chan *domain.Message, 1)}
	roomA := NewRoom("room-1", &nopMsgRepo{}, ps, slog.Default())
	require.NoError(t, roomA.AddUser(alice))

	bob := &captureUser{id: "bob-id", received: make(chan *domain.Message, 1)}
	roomB := NewRoom("room-1", &nopMsgRepo{}, ps, slog.Default())
	require.NoError(t, roomB.AddUser(bob))

	go roomA.Run(t.Context())
	go roomB.Run(t.Context())
	time.Sleep(100 * time.Millisecond)

	msg := &domain.Message{Message: "sup bob", UserID: bob.id, RoomID: roomA.ID()}
	roomA.Broadcast(t.Context(), msg)

	select {
	case got := <-alice.received:
		assert.Equal(t, msg.Message, got.Message)
	case <-time.After(2 * time.Second):
		t.Fatal("alice did not receive msg")
	}

	select {
	case got := <-bob.received:
		assert.Equal(t, msg.Message, got.Message)
	case <-time.After(2 * time.Millisecond):
		t.Fatal("bob did not receive msg")
	}
}
