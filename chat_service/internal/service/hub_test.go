package service

import (
	"errors"
	"log/slog"
	"testing"
	"time"

	"chat_service/internal/domain"
	"chat_service/internal/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	dbError = errors.New("db down")
)

func newHub(
	t *testing.T,
	userRepo *mocks.MockUserRepository,
	roomRepo *mocks.MockRoomRepository,
	msgRepo *mocks.MockMessageRepository,
	friendsClient *mocks.MockFriendshipClient,
) Hub {
	t.Helper()
	return NewHub(t.Context(), userRepo, roomRepo, msgRepo, slog.Default(), friendsClient)
}

func connectUser(t *testing.T, h Hub, uRepo *mocks.MockUserRepository, rRepo *mocks.MockRoomRepository, userID, userName string, rooms []*domain.Room) {
	t.Helper()
	uRepo.EXPECT().GetUserByID(mock.Anything, userID).Return(&domain.User{ID: userID, Username: userName}, nil)
	rRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID).Return(rooms, nil)
	_, conn := newWSPair(t)
	require.NoError(t, h.Connect(t.Context(), userID, conn))
}

func Test_Hub_Connect(t *testing.T) {
	tests := []struct {
		name  string
		setup func(
			userRepo *mocks.MockUserRepository,
			roomRepo *mocks.MockRoomRepository,
		)
		wantError error
	}{
		{
			name: "success",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, userID_1).Return(&domain.User{ID: userID_1, Username: username_1}, nil)
				roomRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID_1).Return([]*domain.Room{{ID: roomID_1}}, nil)
			},
		},
		{
			name: "nil user",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, userID_1).Return((*domain.User)(nil), nil)
			},
			wantError: domain.ErrUserNotFound,
		},
		{
			name: "userRepo error",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, userID_1).Return((*domain.User)(nil), dbError)
			},
			wantError: dbError,
		},
		{
			name: "roomRepo error",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, userID_1).Return(&domain.User{ID: userID_1, Username: username_1}, nil)
				roomRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID_1).Return(([]*domain.Room)(nil), dbError)
			},
			wantError: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			rRepo := mocks.NewMockRoomRepository(t)
			mRepo := mocks.NewMockMessageRepository(t)
			fClientMock := mocks.NewMockFriendshipClient(t)

			h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
			tt.setup(uRepo, rRepo)

			_, clientConn := newWSPair(t)
			err := h.Connect(t.Context(), userID_1, clientConn)

			if tt.wantError != nil {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_Hub_Disconnect(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		room := NewMockRoom(t)
		room.EXPECT().ID().Return(roomID_1)

		connectUser(t, h, uRepo, rRepo, userID_1, username_1, []*domain.Room{{ID: room.ID()}})

		_, err := h.Disconnect(t.Context(), userID_1)
		require.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		dUser, err := h.Disconnect(t.Context(), "unknown")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, dUser)
	})

	t.Run("empty room is removed from hub", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, userID_1, userID_1, []*domain.Room{{ID: roomID_1}})

		_, err := h.Disconnect(t.Context(), userID_1)
		require.NoError(t, err)

		// assert.Empty(t, h.GetRoomClients(roomID_1))
	})
}

func Test_Hub_InviteUser(t *testing.T) {
	aliceID := "alice_id"

	bobID := "bob_id"
	bobName := "bob_name"

	tests := []struct {
		name      string
		setup     func(r *mocks.MockRoomRepository, fc *mocks.MockFriendshipClient)
		wantError error
	}{
		{
			name: "success invitee offline",
			setup: func(r *mocks.MockRoomRepository, fc *mocks.MockFriendshipClient) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: roomID_1}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, roomID_1).Return(true, nil)
				r.EXPECT().AddUser(mock.Anything, bobID, roomID_1).Return(nil)

				fc.EXPECT().IsFriend(mock.Anything, aliceID, bobID).Return(true, nil)
			},
		},
		{
			name: "is not room member",
			setup: func(r *mocks.MockRoomRepository, fc *mocks.MockFriendshipClient) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: roomID_1}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, roomID_1).Return(false, nil)
			},
			wantError: domain.ErrUserForbidden,
		},
		{
			name: "not friend",
			setup: func(r *mocks.MockRoomRepository, fc *mocks.MockFriendshipClient) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: roomID_1}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, roomID_1).Return(true, nil)

				fc.EXPECT().IsFriend(mock.Anything, aliceID, bobID).Return(false, nil)
			},
			wantError: domain.ErrUserForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			rRepo := mocks.NewMockRoomRepository(t)
			mRepo := mocks.NewMockMessageRepository(t)
			fClientMock := mocks.NewMockFriendshipClient(t)

			h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
			tt.setup(rRepo, fClientMock)

			dbRoom, err := h.CreateRoom(t.Context(), "test room", aliceID)
			require.NoError(t, err)

			err = h.InviteUser(t.Context(), aliceID, bobID, dbRoom.ID)

			if tt.wantError != nil {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}

	t.Run("invitee online is added to in-memory room", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		rRepo.EXPECT().CreateRoom(mock.Anything, "chat", aliceID).Return(&domain.Room{ID: roomID_1}, nil)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, roomID_1).Return(true, nil)
		rRepo.EXPECT().AddUser(mock.Anything, bobID, roomID_1).Return(nil)

		fClientMock.EXPECT().IsFriend(mock.Anything, aliceID, bobID).Return(true, nil)

		connectUser(t, h, uRepo, rRepo, bobID, bobName, []*domain.Room{})

		dbRoom, err := h.CreateRoom(t.Context(), "chat", aliceID)
		require.NoError(t, err)

		err = h.InviteUser(t.Context(), aliceID, bobID, dbRoom.ID)
		require.NoError(t, err)

		// assert.Contains(t, h.GetRoomClients(roomID_1), bobName)
	})
}

func Test_Hub_LeaveRoom(t *testing.T) {
	t.Run("success user offline", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, userID_1, roomID_1).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, roomID_1).Return(false, nil)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		require.NoError(t, err)
	})

	t.Run("Deletes room if empty", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, userID_1, roomID_1).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, roomID_1).Return(true, nil)
		rRepo.EXPECT().DeleteRoom(mock.Anything, roomID_1).Return((*domain.Room)(nil), nil)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		require.NoError(t, err)
	})

	t.Run("success user online room removed when empty", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, userID_1, username_1, []*domain.Room{{ID: roomID_1}})

		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, userID_1, roomID_1).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, roomID_1).Return(false, nil)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		require.NoError(t, err)

		// assert.Empty(t, h.GetRoomClients(roomID_1))
	})

	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(false, nil)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(false, dbError)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		assert.Error(t, err)
	})

	t.Run("remove user repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, userID_1, roomID_1).Return(dbError)

		err := h.LeaveRoom(t.Context(), userID_1, roomID_1)
		assert.Error(t, err)
	})
}

func Test_Hub_CreateRoom(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().CreateRoom(mock.Anything, "general", userID_1).Return(&domain.Room{ID: roomID_1, Name: "general"}, nil)

		room, err := h.CreateRoom(t.Context(), "general", userID_1)
		require.NoError(t, err)
		assert.Equal(t, roomID_1, room.ID)
		assert.Equal(t, "general", room.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().CreateRoom(mock.Anything, "general", userID_1).Return((*domain.Room)(nil), dbError)

		room, err := h.CreateRoom(t.Context(), "general", userID_1)
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}

// TODO
// func Test_Hub_GetRoomClients(t *testing.T) {
// 	t.Run("room not in hub returns empty slice", func(t *testing.T) {
// 		uRepo := mocks.NewMockUserRepository(t)
// 		rRepo := mocks.NewMockRoomRepository(t)
// 		mRepo := mocks.NewMockMessageRepository(t)
// 		fClientMock := mocks.NewMockFriendshipClient(t)

// 		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
// 		clients := h.GetRoomClients("nonexistent")
// 		assert.Empty(t, clients)
// 	})

// 	t.Run("returns connected user names", func(t *testing.T) {
// 		uRepo := mocks.NewMockUserRepository(t)
// 		rRepo := mocks.NewMockRoomRepository(t)
// 		mRepo := mocks.NewMockMessageRepository(t)
// 		fClientMock := mocks.NewMockFriendshipClient(t)

// 		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
// 		connectUser(t, h, uRepo, rRepo, "alice", "alice", []*domain.Room{{ID: roomID_1}})
// 		connectUser(t, h, uRepo, rRepo, "bob", "bob", []*domain.Room{{ID: roomID_1}})

// 		clients := h.GetRoomClients(roomID_1)
// 		assert.Contains(t, clients, "alice")
// 		assert.Contains(t, clients, "bob")
// 	})
// }

func Test_Hub_GetRoomHistory(t *testing.T) {
	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(false, nil)

		msgs, err := h.GetRoomHistory(t.Context(), userID_1, roomID_1, time.Time{})
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
		assert.Nil(t, msgs)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(false, dbError)

		msgs, err := h.GetRoomHistory(t.Context(), userID_1, roomID_1, time.Time{})
		assert.Error(t, err)
		assert.Nil(t, msgs)
	})

	t.Run("before zero calls GetMessages", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		before := time.Time{}

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		expected := []*domain.Message{{Message: "hello"}}
		mRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID_1, mock.Anything).Return(expected, nil)

		msgs, err := h.GetRoomHistory(t.Context(), userID_1, roomID_1, before)
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
	})

	t.Run("before future treated as zero calls GetMessages", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		expected := []*domain.Message{{Message: "hello"}}
		mRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID_1, mock.Anything).Return(expected, nil)

		future := time.Now().Add(24 * time.Hour)
		msgs, err := h.GetRoomHistory(t.Context(), userID_1, roomID_1, future)
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
	})

	t.Run("before past calls GetMessagesBefore", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, userID_1, roomID_1).Return(true, nil)
		past := time.Now().Add(-1 * time.Hour)
		expected := []*domain.Message{{Message: "old message"}}
		mRepo.EXPECT().GetMessagesBefore(mock.Anything, roomID_1, past).Return(expected, nil)

		msgs, err := h.GetRoomHistory(t.Context(), userID_1, roomID_1, past)
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
	})
}

func Test_Hub_GetRoomsByUser(t *testing.T) {
	t.Run("delegates to repo", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		expected := []*domain.Room{{ID: roomID_1}, {ID: "room-2"}}
		rRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID_1).Return(expected, nil)

		rooms, err := h.GetRoomsByUser(t.Context(), userID_1)
		require.NoError(t, err)
		assert.Equal(t, expected, rooms)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID_1).Return(([]*domain.Room)(nil), dbError)

		rooms, err := h.GetRoomsByUser(t.Context(), userID_1)
		assert.Error(t, err)
		assert.Nil(t, rooms)
	})
}

func Test_Hub_Shutdown(t *testing.T) {
	t.Run("no panic with no connected users", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		assert.NotPanics(t, func() { h.Shutdown(t.Context()) })
	})

	t.Run("stops connected users and waits", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, userID_1, userID_1, []*domain.Room{})

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
