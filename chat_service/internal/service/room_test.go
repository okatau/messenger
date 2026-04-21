package service

import (
	"context"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type msgRepoMock struct{ mock.Mock }

func (r *msgRepoMock) WriteMessage(ctx context.Context, message *domain.Message) error {
	return nil
}
func (r *msgRepoMock) GetMessages(ctx context.Context, roomID string) ([]*domain.Message, error) {
	args := r.Called(ctx, roomID)
	return args.Get(0).([]*domain.Message), nil
}
func (r *msgRepoMock) GetMessagesBefore(ctx context.Context, roomID string, before time.Time) ([]*domain.Message, error) {
	args := r.Called(ctx, roomID, before)
	return args.Get(0).([]*domain.Message), nil
}

type mockUser struct {
	id          string
	name        string
	outgoingMsg chan *domain.Message
}

func newMockUser(id, name string) *mockUser {
	return &mockUser{
		id:          id,
		name:        name,
		outgoingMsg: make(chan *domain.Message, 1),
	}
}

func (m *mockUser) DeleteRoom(room Room) error { return nil }
func (m *mockUser) Write(ctx context.Context, msg *domain.Message) error {
	m.outgoingMsg <- msg
	return nil
}
func (m *mockUser) Listen(ctx context.Context, wg *sync.WaitGroup) {}
func (m *mockUser) ID() string                                     { return m.id }
func (m *mockUser) Name() string                                   { return m.name }
func (m *mockUser) Rooms() map[string]Room                         { return nil }
func (m *mockUser) Stop()                                          {}

func (m *mockUser) AddRoom(room Room) error { return nil }

var _ repository.MessageRepository = (*msgRepoMock)(nil)

func newTestRoom(id string, repo *msgRepoMock) Room {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRoom(id, repo, logger)
}

func Test_Room_NewRoom(t *testing.T) {
	repo := &msgRepoMock{}
	room := newTestRoom("room-1", repo)
	assert.Equal(t, "room-1", room.ID())
}

func Test_Room_AddUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := newMockUser("user-1", "user-1")
		room := newTestRoom("room-1", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))
		assert.Contains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user exists", func(t *testing.T) {
		user := newMockUser("user-1", "user-1")
		room := newTestRoom("room-1", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))

		err := room.AddUser(user)
		assert.ErrorIs(t, err, domain.ErrUserExists)
	})
}

func Test_Room_RemoveUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := newMockUser("user-1", "user-1")
		room := newTestRoom("room-1", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))

		require.NoError(t, room.RemoveUser(user))
		assert.NotContains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user not found", func(t *testing.T) {
		user := newMockUser("user-1", "user-1")
		room := newTestRoom("room-1", &msgRepoMock{})

		err := room.RemoveUser(user)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func Test_Room_GetUsernames_ReturnsCopy(t *testing.T) {
	user := newMockUser("user", "user")
	room := newTestRoom("room", &msgRepoMock{})

	require.NoError(t, room.AddUser(user))

	list := room.GetUsernames()
	list = list[:len(list)-1]

	assert.Contains(t, room.GetUsernames(), "user")
}

func Test_Room_Broadcast(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := newMockUser("user", "user")
		room := newTestRoom("room", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		defer room.Stop()

		room.Broadcast(t.Context(), &domain.Message{Message: "broadcast success"})

		msg := <-user.outgoingMsg
		assert.Equal(t, "broadcast success", msg.Message)
	})

	t.Run("stooped by stopping room", func(t *testing.T) {
		user := newMockUser("user", "user")
		room := newTestRoom("room", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		room.Stop()

		room.Broadcast(t.Context(), &domain.Message{Message: "should be dropped"})

		select {
		case <-user.outgoingMsg:
			t.Fatal("message should not have been delivered to stopped room")
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("stopping by context cancellation", func(t *testing.T) {
		user := newMockUser("user", "user")
		room := newTestRoom("room", &msgRepoMock{})

		require.NoError(t, room.AddUser(user))

		ctx, cancel := context.WithCancel(t.Context())
		go room.Run(ctx)
		cancel()

		room.Broadcast(ctx, &domain.Message{Message: "should be dropped"})

		select {
		case <-user.outgoingMsg:
			t.Fatal("message should not have been delivered to stopped room")
		case <-time.After(50 * time.Millisecond):
		}
	})
}
