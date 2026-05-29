package service

import (
	"chat_service/internal/domain"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"
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

var _ repository.MessageRepository = (*nopMsgRepo)(nil)

func Test_CrossInstance_MessageDelivery(t *testing.T) {
	rdb := startRedis(t)
	ps := pubsub.NewPubSub(rdb)

	srvConn1, clientConn1 := newWSPair(t)
	srvConn2, clientConn2 := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(aliceID).Return((User)(nil), nil)
	hub.EXPECT().Disconnect(bobID).Return((User)(nil), nil)
	mRepo := &nopMsgRepo{}

	alice := NewUser(aliceID, alice, srvConn1, ps, hub, slog.Default(), mRepo)
	bob := NewUser(bobID, bob, srvConn2, ps, hub, slog.Default(), mRepo)

	alice.AddRoomSub(t.Context(), room1_ID)
	bob.AddRoomSub(t.Context(), room1_ID)

	var awg sync.WaitGroup
	alice.Listen(t.Context(), &awg)

	var bwg sync.WaitGroup
	bob.Listen(t.Context(), &bwg)

	time.Sleep(100 * time.Millisecond)

	msg := &domain.Message{Username: bob.Name(), Message: "Hello, Alice!", RoomID: room1_ID}
	require.NoError(t, clientConn1.WriteJSON(msg))

	var got domain.Message
	require.NoError(t, clientConn2.SetReadDeadline(time.Now().Add(3*time.Second)))
	require.NoError(t, clientConn2.ReadJSON(&got))

	assert.Equal(t, "Hello, Alice!", got.Message)

	clientConn1.Close()
	clientConn2.Close()
	waitDone(t, &awg)
	waitDone(t, &bwg)
}
