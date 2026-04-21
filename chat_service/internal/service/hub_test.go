package service

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var dbError = errors.New("db down")

type userRepoMock struct{ mock.Mock }

func (r *userRepoMock) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	args := r.Called(ctx, userID)
	return args.Get(0).(*domain.User), args.Error(1)
}

type roomRepoMock struct{ mock.Mock }

func (r *roomRepoMock) GetAllRooms(ctx context.Context) ([]*domain.Room, error) {
	return nil, nil
}

func (r *roomRepoMock) GetRoomsByUserID(ctx context.Context, userID string) ([]*domain.Room, error) {
	args := r.Called(ctx, userID)
	return args.Get(0).([]*domain.Room), args.Error(1)
}

func (r *roomRepoMock) GetUsersByRoomID(ctx context.Context, roomID string) ([]*domain.User, error) {
	return nil, nil
}

func (r *roomRepoMock) CreateRoom(ctx context.Context, name, userID string) (*domain.Room, error) {
	args := r.Called(ctx, name, userID)
	return args.Get(0).(*domain.Room), args.Error(1)
}

func (r *roomRepoMock) DeleteRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	return nil, nil
}

func (r *roomRepoMock) AddUser(ctx context.Context, userID, roomID string) error {
	args := r.Called(ctx, userID, roomID)
	return args.Error(0)
}

func (r *roomRepoMock) RemoveUser(ctx context.Context, userID, roomID string) error {
	args := r.Called(ctx, userID, roomID)
	return args.Error(0)
}

func (r *roomRepoMock) IsMember(ctx context.Context, userID, roomID string) (bool, error) {
	args := r.Called(ctx, userID, roomID)
	return args.Get(0).(bool), args.Error(1)
}

func (r *roomRepoMock) IsEmpty(ctx context.Context, roomID string) (bool, error) { return false, nil }

var _ repository.RoomRepository = (*roomRepoMock)(nil)

func newHub(
	t *testing.T,
	userRepo *userRepoMock,
	roomRepo *roomRepoMock,
	msgRepo *msgRepoMock,
) Hub {
	t.Helper()
	return NewHub(t.Context(), userRepo, roomRepo, msgRepo, slog.Default())
}

func connectUser(t *testing.T, h Hub, uRepo *userRepoMock, rRepo *roomRepoMock, userID, userName string, rooms []*domain.Room) {
	t.Helper()
	uRepo.On("GetUserByID", mock.Anything, userID).Return(&domain.User{ID: userID, Username: userName}, nil)
	rRepo.On("GetRoomsByUserID", mock.Anything, userID).Return(rooms, nil)
	_, conn := newWSPair(t)
	require.NoError(t, h.Connect(t.Context(), userID, conn))
}

func Test_Hub_Connect(t *testing.T) {
	user := newMockUser("user", "user")
	tests := []struct {
		name      string
		setup     func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock)
		wantError error
	}{
		{
			name: "success",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				u.On("GetUserByID", mock.Anything, user.ID()).Return(&domain.User{ID: user.ID(), Username: user.Name()}, nil)
				r.On("GetRoomsByUserID", mock.Anything, user.ID()).Return([]*domain.Room{{ID: "room-1"}}, nil)
			},
		},
		{
			name: "nil user",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				u.On("GetUserByID", mock.Anything, user.ID()).Return((*domain.User)(nil), nil)
			},
			wantError: domain.ErrUserNotFound,
		},
		{
			name: "userRepo error",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				u.On("GetUserByID", mock.Anything, user.ID()).Return((*domain.User)(nil), dbError)
			},
			wantError: dbError,
		},
		{
			name: "roomRepo error",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				u.On("GetUserByID", mock.Anything, user.ID()).Return(&domain.User{ID: user.ID(), Username: user.Name()}, nil)
				r.On("GetRoomsByUserID", mock.Anything, user.ID()).Return(([]*domain.Room)(nil), dbError)
			},
			wantError: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			rRepo := &roomRepoMock{}
			mRepo := &msgRepoMock{}

			h := newHub(t, uRepo, rRepo, mRepo)
			tt.setup(uRepo, rRepo, mRepo)

			_, clientConn := newWSPair(t)
			err := h.Connect(t.Context(), user.ID(), clientConn)

			if tt.wantError != nil {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			uRepo.AssertExpectations(t)
			rRepo.AssertExpectations(t)
			mRepo.AssertExpectations(t)
		})
	}
}

func Test_Hub_Disconnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		room := newMockRoom("room-1")
		connectUser(t, h, uRepo, rRepo, "user", "user", []*domain.Room{{ID: room.ID()}})

		dUser, err := h.Disconnect(t.Context(), "user")
		require.NoError(t, err)
		assert.NotContains(t, h.GetRoomClients(room.ID()), dUser.Name())
	})

	t.Run("user not found", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)

		dUser, err := h.Disconnect(t.Context(), "unknown")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, dUser)
	})

	t.Run("empty room is removed from hub", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		connectUser(t, h, uRepo, rRepo, "user", "user", []*domain.Room{{ID: "room-1"}})

		_, err := h.Disconnect(t.Context(), "user")
		require.NoError(t, err)

		assert.Empty(t, h.GetRoomClients("room-1"))
	})
}

func Test_Hub_InviteUser(t *testing.T) {
	alice := newMockUser("alice", "alice")
	bob := newMockUser("bob", "bob")
	room := newMockRoom("room")

	tests := []struct {
		name      string
		setup     func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock)
		wantError error
	}{
		{
			name: "success invitee offline",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				r.On("CreateRoom", mock.Anything, mock.Anything, alice.id).Return(&domain.Room{ID: room.id}, nil)
				r.On("IsMember", mock.Anything, alice.id, room.id).Return(true, nil)
				r.On("AddUser", mock.Anything, bob.id, room.id).Return(nil)
			},
		},
		{
			name: "forbidden",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				r.On("CreateRoom", mock.Anything, mock.Anything, alice.id).Return(&domain.Room{ID: room.id}, nil)
				r.On("IsMember", mock.Anything, alice.id, room.id).Return(false, nil)
			},
			wantError: domain.ErrUserForbidden,
		},
		{
			name: "isMember error",
			setup: func(u *userRepoMock, r *roomRepoMock, m *msgRepoMock) {
				r.On("CreateRoom", mock.Anything, mock.Anything, alice.id).Return(&domain.Room{ID: room.id}, nil)
				r.On("IsMember", mock.Anything, alice.id, room.id).Return(false, dbError)
			},
			wantError: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			rRepo := &roomRepoMock{}
			mRepo := &msgRepoMock{}

			h := newHub(t, uRepo, rRepo, mRepo)
			tt.setup(uRepo, rRepo, mRepo)

			dbRoom, err := h.CreateRoom(t.Context(), "test room", alice.id)
			require.NoError(t, err)

			err = h.InviteUser(t.Context(), alice.id, bob.id, dbRoom.ID)

			if tt.wantError != nil {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			uRepo.AssertExpectations(t)
			rRepo.AssertExpectations(t)
			mRepo.AssertExpectations(t)
		})
	}

	t.Run("invitee online is added to in-memory room", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)

		rRepo.On("CreateRoom", mock.Anything, "chat", alice.id).Return(&domain.Room{ID: "r1"}, nil)
		connectUser(t, h, uRepo, rRepo, bob.id, bob.name, []*domain.Room{})

		rRepo.On("IsMember", mock.Anything, alice.id, "r1").Return(true, nil)
		rRepo.On("AddUser", mock.Anything, bob.id, "r1").Return(nil)

		dbRoom, err := h.CreateRoom(t.Context(), "chat", alice.id)
		require.NoError(t, err)

		err = h.InviteUser(t.Context(), alice.id, bob.id, dbRoom.ID)
		require.NoError(t, err)

		assert.Contains(t, h.GetRoomClients("r1"), bob.name)
	})
}

func Test_Hub_LeaveRoom(t *testing.T) {
	t.Run("success user offline", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		rRepo.On("RemoveUser", mock.Anything, "user", "room-1").Return(nil)

		err := h.LeaveRoom(t.Context(), "user", "room-1")
		require.NoError(t, err)

		rRepo.AssertExpectations(t)
	})

	t.Run("success user online room removed when empty", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		connectUser(t, h, uRepo, rRepo, "user", "user", []*domain.Room{{ID: "room-1"}})

		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		rRepo.On("RemoveUser", mock.Anything, "user", "room-1").Return(nil)

		err := h.LeaveRoom(t.Context(), "user", "room-1")
		require.NoError(t, err)

		assert.Empty(t, h.GetRoomClients("room-1"))
		rRepo.AssertExpectations(t)
	})

	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(false, nil)

		err := h.LeaveRoom(t.Context(), "user", "room-1")
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(false, dbError)

		err := h.LeaveRoom(t.Context(), "user", "room-1")
		assert.Error(t, err)
	})

	t.Run("remove user repo error", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		rRepo.On("RemoveUser", mock.Anything, "user", "room-1").Return(dbError)

		err := h.LeaveRoom(t.Context(), "user", "room-1")
		assert.Error(t, err)
	})
}

func Test_Hub_CreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("CreateRoom", mock.Anything, "general", "user-1").Return(&domain.Room{ID: "r1", Name: "general"}, nil)

		room, err := h.CreateRoom(t.Context(), "general", "user-1")
		require.NoError(t, err)
		assert.Equal(t, "r1", room.ID)
		assert.Equal(t, "general", room.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("CreateRoom", mock.Anything, "general", "user-1").Return((*domain.Room)(nil), dbError)

		room, err := h.CreateRoom(t.Context(), "general", "user-1")
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}

func Test_Hub_GetRoomClients(t *testing.T) {
	t.Run("room not in hub returns empty slice", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		clients := h.GetRoomClients("nonexistent")
		assert.Empty(t, clients)
	})

	t.Run("returns connected user names", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		connectUser(t, h, uRepo, rRepo, "alice", "alice", []*domain.Room{{ID: "room-1"}})
		connectUser(t, h, uRepo, rRepo, "bob", "bob", []*domain.Room{{ID: "room-1"}})

		clients := h.GetRoomClients("room-1")
		assert.Contains(t, clients, "alice")
		assert.Contains(t, clients, "bob")
	})
}

func Test_Hub_GetRoomHistory(t *testing.T) {
	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(false, nil)

		msgs, err := h.GetRoomHistory(t.Context(), "user", "room-1", time.Time{})
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
		assert.Nil(t, msgs)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(false, dbError)

		msgs, err := h.GetRoomHistory(t.Context(), "user", "room-1", time.Time{})
		assert.Error(t, err)
		assert.Nil(t, msgs)
	})

	t.Run("before zero calls GetMessages", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		expected := []*domain.Message{{Message: "hello"}}
		mRepo.On("GetMessages", mock.Anything, "room-1").Return(expected)

		msgs, err := h.GetRoomHistory(t.Context(), "user", "room-1", time.Time{})
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
		mRepo.AssertExpectations(t)
	})

	t.Run("before future treated as zero calls GetMessages", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		expected := []*domain.Message{{Message: "hello"}}
		mRepo.On("GetMessages", mock.Anything, "room-1").Return(expected)

		future := time.Now().Add(24 * time.Hour)
		msgs, err := h.GetRoomHistory(t.Context(), "user", "room-1", future)
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
		mRepo.AssertExpectations(t)
	})

	t.Run("before past calls GetMessagesBefore", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("IsMember", mock.Anything, "user", "room-1").Return(true, nil)
		past := time.Now().Add(-1 * time.Hour)
		expected := []*domain.Message{{Message: "old message"}}
		mRepo.On("GetMessagesBefore", mock.Anything, "room-1", past).Return(expected)

		msgs, err := h.GetRoomHistory(t.Context(), "user", "room-1", past)
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
		mRepo.AssertExpectations(t)
	})
}

func Test_Hub_GetRoomsByUser(t *testing.T) {
	t.Run("delegates to repo", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		expected := []*domain.Room{{ID: "r1"}, {ID: "r2"}}
		rRepo.On("GetRoomsByUserID", mock.Anything, "user-1").Return(expected, nil)

		rooms, err := h.GetRoomsByUser(t.Context(), "user-1")
		require.NoError(t, err)
		assert.Equal(t, expected, rooms)
		rRepo.AssertExpectations(t)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		rRepo.On("GetRoomsByUserID", mock.Anything, "user-1").Return(([]*domain.Room)(nil), dbError)

		rooms, err := h.GetRoomsByUser(t.Context(), "user-1")
		assert.Error(t, err)
		assert.Nil(t, rooms)
	})
}

func Test_Hub_Shutdown(t *testing.T) {
	t.Run("no panic with no connected users", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)

		assert.NotPanics(t, func() { h.Shutdown(t.Context()) })
	})

	t.Run("stops connected users and waits", func(t *testing.T) {
		uRepo := &userRepoMock{}
		rRepo := &roomRepoMock{}
		mRepo := &msgRepoMock{}

		h := newHub(t, uRepo, rRepo, mRepo)
		connectUser(t, h, uRepo, rRepo, "user", "user", []*domain.Room{})

		done := make(chan struct{})
		go func() {
			h.Shutdown(t.Context())
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Shutdown did not complete in time")
		}
	})
}
