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
	"github.com/stretchr/testify/require"
)

type mockRoom struct {
	id          string
	broadcastCh chan *domain.Message
}

func newMockRoom(id string) *mockRoom {
	return &mockRoom{id: id, broadcastCh: make(chan *domain.Message, 1)}
}

func (r *mockRoom) ID() string                                     { return r.id }
func (r *mockRoom) AddUser(_ User) error                           { return nil }
func (r *mockRoom) RemoveUser(_ User) error                        { return nil }
func (r *mockRoom) IsEmpty() bool                                  { return false }
func (r *mockRoom) Broadcast(_ context.Context, m *domain.Message) { r.broadcastCh <- m }
func (r *mockRoom) Run(_ context.Context)                          {}
func (r *mockRoom) Stop()                                          {}
func (r *mockRoom) GetUsernames() []string                         { return nil }

type mockHub struct {
	mu              sync.Mutex
	disconnectedIDs []string
}

func (h *mockHub) Connect(_ context.Context, _ string, _ *ws.Conn) error { return nil }
func (h *mockHub) Disconnect(_ context.Context, id string) (User, error) {
	h.mu.Lock()
	h.disconnectedIDs = append(h.disconnectedIDs, id)
	h.mu.Unlock()
	return nil, nil
}
func (h *mockHub) InviteUser(_ context.Context, _, _, _ string) error              { return nil }
func (h *mockHub) LeaveRoom(_ context.Context, _, _ string) error                  { return nil }
func (h *mockHub) Shutdown(_ context.Context)                                      {}
func (h *mockHub) CreateRoom(_ context.Context, _, _ string) (*domain.Room, error) { return nil, nil }
func (h *mockHub) GetRoomClients(_ string) []string                                { return nil }
func (h *mockHub) GetRoomHistory(_ context.Context, _, _ string, _ time.Time) ([]*domain.Message, error) {
	return nil, nil
}
func (h *mockHub) GetRoomsByUser(_ context.Context, _ string) ([]*domain.Room, error) {
	return nil, nil
}

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

// ─── тесты геттеров ─────────────────────────────────────────────────────────

func TestUser_ID(t *testing.T) {
	u := newTestUser("user-1", "alice", nil, &mockHub{})
	assert.Equal(t, "user-1", u.ID())
}

func TestUser_Name(t *testing.T) {
	u := newTestUser("user-1", "alice", nil, &mockHub{})
	assert.Equal(t, "alice", u.Name())
}

// ─── тесты AddRoom ──────────────────────────────────────────────────────────

func TestUser_AddRoom(t *testing.T) {
	t.Run("успешное добавление", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})
		room := newMockRoom("room-1")

		require.NoError(t, u.AddRoom(room))

		assert.Contains(t, u.Rooms(), "room-1")
	})

	t.Run("дублирующая комната возвращает ErrRoomExists", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})
		room := newMockRoom("room-1")
		require.NoError(t, u.AddRoom(room))

		err := u.AddRoom(room)

		assert.ErrorIs(t, err, domain.ErrRoomExists)
	})
}

// ─── тесты DeleteRoom ───────────────────────────────────────────────────────

func TestUser_DeleteRoom(t *testing.T) {
	t.Run("успешное удаление", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})
		room := newMockRoom("room-1")
		require.NoError(t, u.AddRoom(room))

		require.NoError(t, u.DeleteRoom(room))

		assert.NotContains(t, u.Rooms(), "room-1")
	})

	t.Run("несуществующая комната возвращает ErrRoomNotFound", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})

		err := u.DeleteRoom(newMockRoom("room-1"))

		assert.ErrorIs(t, err, domain.ErrRoomNotFound)
	})
}

// ─── тесты Rooms ────────────────────────────────────────────────────────────

func TestUser_Rooms_ReturnsCopy(t *testing.T) {
	u := newTestUser("u1", "alice", nil, &mockHub{})
	require.NoError(t, u.AddRoom(newMockRoom("room-1")))

	rooms := u.Rooms()
	delete(rooms, "room-1") // мутируем копию

	// оригинал не должен измениться
	assert.Contains(t, u.Rooms(), "room-1")
}

// ─── тесты Write ────────────────────────────────────────────────────────────

func TestUser_Write(t *testing.T) {
	t.Run("сообщение помещается в буфер", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})

		err := u.Write(context.Background(), &domain.Message{Message: "hello"})

		assert.NoError(t, err)
	})

	t.Run("переполнение буфера возвращает ErrUserDisconnected", func(t *testing.T) {
		u := newTestUser("u1", "alice", nil, &mockHub{})
		msg := &domain.Message{Message: "x"}

		// заполняем буфер до упора
		for i := 0; i < MaxBufSize; i++ {
			require.NoError(t, u.Write(context.Background(), msg))
		}

		err := u.Write(context.Background(), msg)

		assert.ErrorIs(t, err, domain.ErrUserDisconnected)
	})
}

// ─── тесты горутин Listen ───────────────────────────────────────────────────

// TestUser_ListenWrite проверяет, что listenWrite доставляет сообщение клиенту.
func TestUser_ListenWrite_DeliversMsgToClient(t *testing.T) {
	serverConn, clientConn := newWSPair(t)
	u := newTestUser("u1", "alice", serverConn, &mockHub{})

	// кладём сообщение в канал до старта горутин
	msg := &domain.Message{Message: "ping", RoomID: "room-1"}
	require.NoError(t, u.Write(context.Background(), msg))

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	var got domain.Message
	require.NoError(t, clientConn.SetReadDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, clientConn.ReadJSON(&got))

	assert.Equal(t, "ping", got.Message)
	assert.Equal(t, "room-1", got.RoomID)

	// закрываем клиент → ошибка чтения на сервере → doneCh закрывается → обе горутины выходят
	clientConn.Close()
	waitDone(t, &wg)
}

// TestUser_ListenRead_BroadcastsToRoom проверяет, что listenRead читает сообщение
// от клиента и вызывает Broadcast нужной комнаты с обогащёнными полями.
func TestUser_ListenRead_BroadcastsToRoom(t *testing.T) {
	serverConn, clientConn := newWSPair(t)
	hub := &mockHub{}
	u := newTestUser("u1", "alice", serverConn, hub)

	room := newMockRoom("room-1")
	require.NoError(t, u.AddRoom(room))

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	outMsg := domain.Message{RoomID: "room-1", Message: "hello from client"}
	require.NoError(t, clientConn.SetWriteDeadline(time.Now().Add(2*time.Second)))
	require.NoError(t, clientConn.WriteJSON(outMsg))

	select {
	case received := <-room.broadcastCh:
		assert.Equal(t, "hello from client", received.Message)
		assert.Equal(t, "room-1", received.RoomID)
		assert.Equal(t, "u1", received.UserID)      // проставляется сервером
		assert.Equal(t, "alice", received.Username) // проставляется сервером
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
	hub := &mockHub{}
	u := newTestUser("u1", "alice", serverConn, hub)

	var wg sync.WaitGroup
	u.Listen(context.Background(), &wg)

	clientConn.Close()
	waitDone(t, &wg)

	hub.mu.Lock()
	defer hub.mu.Unlock()
	assert.Contains(t, hub.disconnectedIDs, "u1")
}
