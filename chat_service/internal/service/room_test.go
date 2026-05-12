package service

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/mocks"
	"chat_service/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	roomID_1   = "room-1"
	userID_1   = "user-1"
	username_1 = "username"
)

func newTestRoom(id string, repo repository.MessageRepository) Room {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRoom(id, repo, logger)
}

func Test_Room_NewRoom(t *testing.T) {
	repo := mocks.NewMockMessageRepository(t)
	room := newTestRoom(roomID_1, repo)
	assert.Equal(t, roomID_1, room.ID())
}

func Test_Room_AddUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)
		user.EXPECT().Name().Return(username_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

		require.NoError(t, room.AddUser(user))
		assert.Contains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user exists", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

		require.NoError(t, room.AddUser(user))

		err := room.AddUser(user)
		assert.ErrorIs(t, err, domain.ErrUserExists)
	})
}

func Test_Room_RemoveUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)
		user.EXPECT().Name().Return(username_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

		require.NoError(t, room.AddUser(user))

		require.NoError(t, room.RemoveUser(user))
		assert.NotContains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user not found", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

		err := room.RemoveUser(user)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func Test_Room_GetUsernames_ReturnsCopy(t *testing.T) {
	user := NewMockUser(t)
	user.EXPECT().ID().Return(userID_1)
	user.EXPECT().Name().Return(username_1)

	room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

	require.NoError(t, room.AddUser(user))

	list := room.GetUsernames()
	list = list[:len(list)-1]

	assert.Contains(t, room.GetUsernames(), username_1)
}

func Test_Room_Broadcast(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		msg := &domain.Message{Message: "broadcast success"}

		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		broadcastCh := make(chan *domain.Message)
		user.EXPECT().Write(mock.Anything, msg).
			Run(func(_ context.Context, msg *domain.Message) {
				broadcastCh <- msg
			}).Return(nil)

		mrepo := mocks.NewMockMessageRepository(t)
		writeCh := make(chan struct{})
		mrepo.EXPECT().WriteMessage(mock.Anything, mock.Anything).
			Run(func(_ context.Context, _ *domain.Message) {
				close(writeCh)
			}).Return(nil)

		room := newTestRoom(roomID_1, mrepo)
		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		defer room.Stop()

		room.Broadcast(t.Context(), msg)
		receivedMsg := <-broadcastCh
		assert.Equal(t, "broadcast success", receivedMsg.Message)

		select {
		case <-writeCh:
		case <-time.After(time.Second):
			t.Fatal("WriteMessage was not called")
		}
	})

	t.Run("stooped by stopping room", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))

		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		room.Stop()

		room.Broadcast(t.Context(), &domain.Message{Message: "should be dropped"})

		select {
		case <-time.After(50 * time.Millisecond):
		}
	})

	t.Run("stopping by context cancellation", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t))
		require.NoError(t, room.AddUser(user))

		ctx, cancel := context.WithCancel(t.Context())
		go room.Run(ctx)
		cancel()

		room.Broadcast(ctx, &domain.Message{Message: "should be dropped"})

		select {
		case <-time.After(50 * time.Millisecond):
		}
	})
}
