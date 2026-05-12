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

	ws "github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var wsUpgrader = ws.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

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

func newTestUser(id, name string, conn *ws.Conn, hub Hub) User {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewUser(id, name, conn, hub, logger)
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
	u := newTestUser("user-1", "alice", nil, NewMockHub(t))
	assert.Equal(t, "user-1", u.ID())
}

func TestUser_Name(t *testing.T) {
	u := newTestUser("user-1", "alice", nil, NewMockHub(t))
	assert.Equal(t, "alice", u.Name())
}

func TestUser_AddRoom(t *testing.T) {
	t.Run("success adding room", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))

		room := NewMockRoom(t)
		roomID := "room-id"
		room.EXPECT().ID().Return(roomID)

		require.NoError(t, u.AddRoom(room))

		assert.Contains(t, u.Rooms(), roomID)
	})

	t.Run("error adding already existing room", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))

		room := NewMockRoom(t)
		roomID := "room-id"
		room.EXPECT().ID().Return(roomID)

		require.NoError(t, u.AddRoom(room))
		err := u.AddRoom(room)
		assert.ErrorIs(t, err, domain.ErrRoomExists)
	})
}

func TestUser_DeleteRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))

		room := NewMockRoom(t)
		roomID := "room-id"
		room.EXPECT().ID().Return(roomID)

		require.NoError(t, u.AddRoom(room))

		require.NoError(t, u.DeleteRoom(room))
		assert.NotContains(t, u.Rooms(), roomID)
	})

	t.Run("error romm does not exists", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))

		room := NewMockRoom(t)
		roomID := "room-id"
		room.EXPECT().ID().Return(roomID)

		err := u.DeleteRoom(room)

		assert.ErrorIs(t, err, domain.ErrRoomNotFound)
	})
}

func TestUser_Rooms_ReturnsCopy(t *testing.T) {
	u := newTestUser("u1", "alice", nil, NewMockHub(t))

	room := NewMockRoom(t)
	roomID := "room-id"
	room.EXPECT().ID().Return(roomID)

	require.NoError(t, u.AddRoom(room))

	rooms := u.Rooms()
	delete(rooms, roomID)

	assert.Contains(t, u.Rooms(), roomID)
}

func TestUser_Write(t *testing.T) {
	t.Run("write message to outgoindMsg", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))

		err := u.Write(context.Background(), &domain.Message{Message: "hello"})

		assert.NoError(t, err)
	})

	t.Run("error outgoindMsg overflow", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, NewMockHub(t))
		msg := &domain.Message{Message: "x"}

		for i := 0; i < MaxBufSize; i++ {
			require.NoError(t, u.Write(context.Background(), msg))
		}

		err := u.Write(context.Background(), msg)

		assert.ErrorIs(t, err, domain.ErrUserDisconnected)
	})
}

// TestUser_ListenWrite проверяет, что listenWrite доставляет сообщение клиенту.
func TestUser_ListenWrite_DeliversMsgToClient(t *testing.T) {
	serverConn, clientConn := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(mock.Anything, "u1").Return(nil, nil)

	u := newTestUser("u1", "alice", serverConn, hub)

	roomID := "room-id"

	msg := &domain.Message{Message: "ping", RoomID: roomID}
	require.NoError(t, u.Write(context.Background(), msg))

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	var got domain.Message
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, clientConn.ReadJSON(&got))

	assert.Equal(t, "ping", got.Message)
	assert.Equal(t, roomID, got.RoomID)

	clientConn.Close()
	waitDone(t, &wg)
}

func TestUser_ListenRead_BroadcastsToRoom(t *testing.T) {
	serverConn, clientConn := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(mock.Anything, "u1").Return(nil, nil)

	u := newTestUser("u1", "alice", serverConn, hub)

	room := NewMockRoom(t)
	roomID := "room-id"
	room.EXPECT().ID().Return(roomID)

	broadcastCh := make(chan *domain.Message, 1)
	room.EXPECT().Broadcast(mock.Anything, mock.Anything).
		Run(func(_ context.Context, msg *domain.Message) {
			broadcastCh <- msg
		})

	require.NoError(t, u.AddRoom(room))

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	outMsg := domain.Message{RoomID: roomID, Message: "hello from client"}
	require.NoError(t, clientConn.SetWriteDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, clientConn.WriteJSON(outMsg))

	select {
	case received := <-broadcastCh:
		assert.Equal(t, "hello from client", received.Message)
		assert.Equal(t, roomID, received.RoomID)
		assert.Equal(t, "u1", received.UserID)
		assert.Equal(t, "alice", received.Username)
		assert.False(t, received.Timestamp.IsZero())
	case <-time.After(2 * time.Second):
		t.Fatal("timeout: broadcast was not called")
	}

	clientConn.Close()
	waitDone(t, &wg)
}

// TestUser_ListenRead_ClosedConnCallsDisconnect проверяет, что при разрыве
// соединения вызывается hub.Disconnect с корректным userID.
func TestUser_ListenRead_ClosedConnCallsDisconnect(t *testing.T) {
	serverConn, clientConn := newWSPair(t)

	hub := NewMockHub(t)
	hub.EXPECT().Disconnect(mock.Anything, "u1").Return(nil, nil)

	u := newTestUser("u1", "alice", serverConn, hub)

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	clientConn.Close()
	waitDone(t, &wg)
}
