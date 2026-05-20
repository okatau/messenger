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

func newTestRoom(id string, repo repository.MessageRepository, ps *mocks.MockPubSub) Room {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	return NewRoom(id, repo, ps, logger)
}

func Test_Room_NewRoom(t *testing.T) {
	repo := mocks.NewMockMessageRepository(t)
	ps := mocks.NewMockPubSub(t)
	room := newTestRoom(roomID_1, repo, ps)
	assert.Equal(t, roomID_1, room.ID())
}

func Test_Room_AddUser(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)
		user.EXPECT().Name().Return(username_1)

		room := newTestRoom(roomID_1, nil, nil)

		require.NoError(t, room.AddUser(user))
		assert.Contains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user exists", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, nil, nil)

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

		room := newTestRoom(roomID_1, nil, nil)

		require.NoError(t, room.AddUser(user))

		require.NoError(t, room.RemoveUser(user))
		assert.NotContains(t, room.GetUsernames(), user.Name())
	})

	t.Run("user not found", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		room := newTestRoom(roomID_1, nil, nil)

		err := room.RemoveUser(user)
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
	})
}

func Test_Room_GetUsernames_ReturnsCopy(t *testing.T) {
	user := NewMockUser(t)
	user.EXPECT().ID().Return(userID_1)
	user.EXPECT().Name().Return(username_1)

	room := newTestRoom(roomID_1, nil, nil)

	require.NoError(t, room.AddUser(user))

	list := room.GetUsernames()
	list = list[:len(list)-1]

	assert.Contains(t, room.GetUsernames(), username_1)
}
func makeSubscribedPS(t *testing.T, channel string) (*mocks.MockPubSub, chan *domain.Message) {
	t.Helper()
	ps := mocks.NewMockPubSub(t)
	msgCh := make(chan *domain.Message, 1)
	ps.EXPECT().Subscribe(mock.Anything, channel).Return(msgCh, func() {}).Maybe()

	return ps, msgCh
}

func Test_Room_Broadcast(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		msg := &domain.Message{Message: "broadcast success"}

		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		broadcastCh := make(chan *domain.Message, 1)
		user.EXPECT().Write(mock.Anything, msg).
			Run(func(_ context.Context, m *domain.Message) {
				broadcastCh <- m
			}).Return(nil)

		mrepo := mocks.NewMockMessageRepository(t)
		mrepo.EXPECT().WriteMessage(mock.Anything, mock.Anything).Return(nil)

		channel := "room:" + roomID_1

		ps, msgCh := makeSubscribedPS(t, channel)
		ps.EXPECT().Publish(mock.Anything, channel, msg).
			Run(func(_ context.Context, _ string, m *domain.Message) {
				msgCh <- m
			}).Return(nil)

		room := newTestRoom(roomID_1, mrepo, ps)
		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		defer room.Stop()

		room.Broadcast(t.Context(), msg)

		select {
		case receivedMsg := <-broadcastCh:
			assert.Equal(t, "broadcast success", receivedMsg.Message)
		case <-time.After(time.Second):
			t.Fatal("user.Write was not called")
		}
	})

	t.Run("stopped by stopping room", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		ps, _ := makeSubscribedPS(t, mock.Anything)
		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t), ps)
		require.NoError(t, room.AddUser(user))

		go room.Run(t.Context())
		room.Stop()

		room.Broadcast(t.Context(), &domain.Message{Message: "should be dropped"})
	})

	t.Run("stopped by context cancellation", func(t *testing.T) {
		user := NewMockUser(t)
		user.EXPECT().ID().Return(userID_1)

		ps, _ := makeSubscribedPS(t, mock.Anything)
		room := newTestRoom(roomID_1, mocks.NewMockMessageRepository(t), ps)
		require.NoError(t, room.AddUser(user))

		ctx, cancel := context.WithCancel(t.Context())
		go room.Run(ctx)
		cancel()

		room.Broadcast(ctx, &domain.Message{Message: "should be dropped"})
	})
}
