package pubsub

import (
	"chat_service/internal/domain"
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

func Test_RoundTrip(t *testing.T) {
	rdb := startRedis(t)
	ps := NewPubSub(rdb)

	roomID := "room-1"
	chanName := "room:" + roomID
	msg := &domain.Message{Message: "hello", RoomID: roomID, UserID: "user-1"}

	msgCh, unsub := ps.Subscribe(t.Context(), chanName)
	defer unsub()
	time.Sleep(50 * time.Millisecond)

	require.NoError(t, ps.Publish(t.Context(), chanName, msg))

	select {
	case got := <-msgCh:
		assert.Equal(t, msg.Message, got.Message)
		assert.Equal(t, msg.RoomID, got.RoomID)
	case <-time.After(2 * time.Second):
		t.Fatal("message not received")
	}
}

func Test_NewInstance(t *testing.T) {
	rdb := startRedis(t)

	ps1 := NewPubSub(rdb)
	ps2 := NewPubSub(rdb)

	roomID := "room-42"
	chanName := "room:" + roomID

	ch1, unsub1 := ps1.Subscribe(t.Context(), chanName)
	defer unsub1()
	ch2, unsub2 := ps2.Subscribe(t.Context(), chanName)
	defer unsub2()

	time.Sleep(50 * time.Millisecond)

	msg := &domain.Message{Message: "cross-instance"}
	require.NoError(t, ps1.Publish(t.Context(), chanName, msg))

	for i, ch := range []<-chan *domain.Message{ch1, ch2} {
		select {
		case got := <-ch:
			assert.Equal(t, "cross-instance", got.Message, "subscriber %d", i+1)
		case <-time.After(2 * time.Second):
			t.Fatalf("subscriber %d did not receive message", i+1)
		}
	}

}
