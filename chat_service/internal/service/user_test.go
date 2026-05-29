package service

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/mocks"
	"chat_service/internal/pubsub"
	"chat_service/internal/repository"

	"github.com/google/uuid"
	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var wsUpgrader = ws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

var (
	alice   = "alice"
	aliceID = uuid.NewString()

	bob   = "bob"
	bobID = uuid.NewString()

	room1    = "room1"
	room1_ID = uuid.NewString()

	room2_ID = uuid.NewString()
)

func newWSPair(t *testing.T) (serverConn, clientConn *ws.Conn) {
	t.Helper()

	ready := make(chan *ws.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := wsUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		ready <- c
	}))
	t.Cleanup(srv.Close)

	clientConn, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	require.NoError(t, err)
	t.Cleanup(func() { clientConn.Close() })

	serverConn = <-ready
	t.Cleanup(func() { serverConn.Close() })
	return
}

func newTestUser(id, name string, conn *ws.Conn, hub Hub, ps pubsub.PubSub, mrepo repository.MessageRepository) User {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewUser(id, name, conn, ps, hub, logger, mrepo)
}

// waitDone ждёт завершения горутин из Listen с таймаутом 3 секунды.
func waitDone(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("goroutines did not stop in time")
	}
}

func TestUser_ID(t *testing.T) {
	u := newTestUser(aliceID, alice, nil, NewMockHub(t), mocks.NewMockPubSub(t), mocks.NewMockMessageRepository(t))
	assert.Equal(t, aliceID, u.ID())
}

func TestUser_Name(t *testing.T) {
	u := newTestUser(aliceID, alice, nil, NewMockHub(t), mocks.NewMockPubSub(t), mocks.NewMockMessageRepository(t))
	assert.Equal(t, alice, u.Name())
}

func TestUser_AddRoomSub(t *testing.T) {
	t.Run("success adding room", func(t *testing.T) {
		ps := mocks.NewMockPubSub(t)
		u := newTestUser(aliceID, alice, nil, NewMockHub(t), ps, mocks.NewMockMessageRepository(t))

		msgCh := make(chan *domain.Message)
		ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(msgCh, func() {})

		u.AddRoomSub(t.Context(), room1_ID)
	})

	t.Run("message from pubsub is delivered to ws client", func(t *testing.T) {
		serverConn, clientConn := newWSPair(t)

		hub := NewMockHub(t)
		hub.EXPECT().Disconnect(aliceID).Return(nil, nil)

		ps := mocks.NewMockPubSub(t)
		u := newTestUser(aliceID, alice, serverConn, hub, ps, mocks.NewMockMessageRepository(t))

		msgCh := make(chan *domain.Message, 1)
		ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(msgCh, func() {})

		u.AddRoomSub(t.Context(), room1_ID)

		var wg sync.WaitGroup
		u.Listen(context.Background(), &wg)

		msgCh <- &domain.Message{Message: "hello from room", RoomID: room1_ID}

		var got domain.Message
		require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(2*time.Second)))
		require.NoError(t, clientConn.ReadJSON(&got))

		assert.Equal(t, "hello from room", got.Message)

		clientConn.Close()
		waitDone(t, &wg)
	})
}

func TestUser_RemoveRoomSub(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ps := mocks.NewMockPubSub(t)
		u := newTestUser(aliceID, alice, nil, NewMockHub(t), ps, mocks.NewMockMessageRepository(t))

		msgCh := make(chan *domain.Message)
		ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(msgCh, func() {})

		u.AddRoomSub(t.Context(), room1_ID)

		require.NoError(t, u.RemoveRoomSub(room1_ID))
	})

	t.Run("room does not exists", func(t *testing.T) {
		ps := mocks.NewMockPubSub(t)
		u := newTestUser(aliceID, alice, nil, NewMockHub(t), ps, mocks.NewMockMessageRepository(t))

		// msgCh := make(chan *domain.Message)
		// ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(msgCh, func() {})

		require.ErrorIs(t, u.RemoveRoomSub(room1_ID), domain.ErrRoomNotFound)
	})
}

// TestUser_ListenWrite проверяет, что listenWrite доставляет сообщение клиенту.
func TestUser_ListenWrite_DeliversMsgToClient(t *testing.T) {
	serverConn, clientConn := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(aliceID).Return(nil, nil)

	ps := mocks.NewMockPubSub(t)
	u := newTestUser(aliceID, alice, serverConn, hub, ps, mocks.NewMockMessageRepository(t))

	msgCh := make(chan *domain.Message, 1)
	ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(msgCh, func() {})

	u.AddRoomSub(t.Context(), room1_ID)

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	msgCh <- &domain.Message{Message: "hello from room", RoomID: room1_ID}

	var got domain.Message
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, clientConn.ReadJSON(&got))

	assert.Equal(t, "hello from room", got.Message)

	clientConn.Close()
	waitDone(t, &wg)
}

// TestUser_ListenRead_ClosedConnCallsDisconnect проверяет, что при разрыве
// соединения вызывается hub.Disconnect с корректным userID.
func TestUser_ListenRead_ClosedConnCallsDisconnect(t *testing.T) {
	serverConn, clientConn := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(aliceID).Return(nil, nil)

	ps := mocks.NewMockPubSub(t)
	ps.EXPECT().Subscribe(mock.Anything, channelKey(room1_ID)).Return(make(chan *domain.Message, 1), func() {})

	u := newTestUser(aliceID, alice, serverConn, hub, ps, mocks.NewMockMessageRepository(t))
	u.AddRoomSub(t.Context(), room1_ID)

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	clientConn.Close()
	waitDone(t, &wg)
}
