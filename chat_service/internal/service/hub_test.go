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
	errDB = errors.New("db down")
)

// newNopPubSub returns a PubSub mock whose Subscribe may be called any number of times.
// Hub tests exercise hub logic, not message routing, so rooms can run with a silent channel.
func newNopPubSub(t *testing.T) *mocks.MockPubSub {
	t.Helper()
	ps := mocks.NewMockPubSub(t)
	ps.EXPECT().Subscribe(mock.Anything, mock.Anything).
		Return(make(chan *domain.Message, 1), func() {}).
		Maybe()
	return ps
}

func newHub(
	t *testing.T,
	userRepo *mocks.MockUserRepository,
	roomRepo *mocks.MockRoomRepository,
	msgRepo *mocks.MockMessageRepository,
	friendsClient *mocks.MockFriendshipClient,
) Hub {
	t.Helper()
	return NewHub(t.Context(), userRepo, roomRepo, msgRepo, slog.Default(), friendsClient, newNopPubSub(t))
}

func connectUser(t *testing.T, h Hub, uRepo *mocks.MockUserRepository, rRepo *mocks.MockRoomRepository, userID, userName string, rooms []*domain.Room) {
	t.Helper()
	uRepo.EXPECT().GetUserByID(mock.Anything, userID).Return(&domain.User{ID: userID, Username: userName}, nil)
	rRepo.EXPECT().GetRoomsByUserID(mock.Anything, userID).Return(rooms, nil)
	rRepo.EXPECT().GetDMsByUserID(mock.Anything, userID).Return([]*domain.Room{}, nil)

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
				userRepo.EXPECT().GetUserByID(mock.Anything, aliceID).Return(&domain.User{ID: aliceID, Username: alice}, nil)
				roomRepo.EXPECT().GetRoomsByUserID(mock.Anything, aliceID).Return([]*domain.Room{{ID: room1_ID}}, nil)
				roomRepo.EXPECT().GetDMsByUserID(mock.Anything, aliceID).Return([]*domain.Room{}, nil)
			},
		},
		{
			name: "nil user",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, aliceID).Return((*domain.User)(nil), nil)
			},
			wantError: domain.ErrUserNotFound,
		},
		{
			name: "userRepo error",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, aliceID).Return((*domain.User)(nil), errDB)
			},
			wantError: errDB,
		},
		{
			name: "GetRoomsByUserID error",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, aliceID).Return(&domain.User{ID: aliceID, Username: alice}, nil)
				roomRepo.EXPECT().GetRoomsByUserID(mock.Anything, aliceID).Return(([]*domain.Room)(nil), errDB)
			},
			wantError: errDB,
		},
		{
			name: "GetDMsByUserID error",
			setup: func(
				userRepo *mocks.MockUserRepository,
				roomRepo *mocks.MockRoomRepository,
			) {
				userRepo.EXPECT().GetUserByID(mock.Anything, aliceID).Return(&domain.User{ID: aliceID, Username: alice}, nil)
				roomRepo.EXPECT().GetRoomsByUserID(mock.Anything, aliceID).Return([]*domain.Room{{ID: room1_ID}}, nil)
				roomRepo.EXPECT().GetDMsByUserID(mock.Anything, aliceID).Return(([]*domain.Room)(nil), errDB)
			},
			wantError: errDB,
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
			err := h.Connect(t.Context(), aliceID, clientConn)

			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
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
		connectUser(t, h, uRepo, rRepo, aliceID, alice, []*domain.Room{{ID: room1_ID}})

		_, err := h.Disconnect(aliceID)
		require.NoError(t, err)
	})

	t.Run("user not found", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		dUser, err := h.Disconnect("unknown")
		assert.ErrorIs(t, err, domain.ErrUserNotFound)
		assert.Nil(t, dUser)
	})

	t.Run("empty room is removed from hub", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, aliceID, alice, []*domain.Room{{ID: room1_ID}})

		_, err := h.Disconnect(aliceID)
		require.NoError(t, err)
	})
}

func Test_Hub_InviteUser(t *testing.T) {
	tests := []struct {
		name  string
		setup func(
			r *mocks.MockRoomRepository,
			u *mocks.MockUserRepository,
		)
		wantError error
	}{
		{
			name: "success invitee offline",
			setup: func(
				r *mocks.MockRoomRepository,
				u *mocks.MockUserRepository,
			) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: room1_ID}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
				u.EXPECT().GetInviteAvailability(mock.Anything, bobID).Return(true, nil)
				r.EXPECT().AddUser(mock.Anything, bobID, room1_ID).Return(nil)
				r.EXPECT().GetRoomType(mock.Anything, room1_ID).Return("group", nil)
			},
		},
		{
			name: "is not room member",
			setup: func(
				r *mocks.MockRoomRepository,
				u *mocks.MockUserRepository,
			) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: room1_ID}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(false, nil)
			},
			wantError: domain.ErrUserForbidden,
		},
		{
			name: "invites not available",
			setup: func(
				r *mocks.MockRoomRepository,
				u *mocks.MockUserRepository,
			) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: room1_ID}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)

				u.EXPECT().GetInviteAvailability(mock.Anything, bobID).Return(false, nil)
			},
			wantError: domain.ErrUserForbidden,
		},
		{
			name: "invalid room type",
			setup: func(
				r *mocks.MockRoomRepository,
				u *mocks.MockUserRepository,
			) {
				r.EXPECT().CreateRoom(mock.Anything, mock.Anything, aliceID).Return(&domain.Room{ID: room1_ID}, nil)
				r.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
				r.EXPECT().GetRoomType(mock.Anything, room1_ID).Return("direct", nil)
				u.EXPECT().GetInviteAvailability(mock.Anything, bobID).Return(true, nil)

				u.EXPECT().GetInviteAvailability(mock.Anything, bobID).Return(false, nil)
			},
			wantError: domain.ErrRoomNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			rRepo := mocks.NewMockRoomRepository(t)
			mRepo := mocks.NewMockMessageRepository(t)
			fClientMock := mocks.NewMockFriendshipClient(t)

			h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
			tt.setup(rRepo, uRepo)

			dbRoom, err := h.CreateRoom(t.Context(), room1, aliceID)
			require.NoError(t, err)

			err = h.InviteUser(t.Context(), aliceID, bobID, dbRoom.ID)

			if tt.wantError != nil {
				assert.ErrorIs(t, err, tt.wantError)
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

		rRepo.EXPECT().CreateRoom(mock.Anything, room1, aliceID).Return(&domain.Room{ID: room1_ID}, nil)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		rRepo.EXPECT().AddUser(mock.Anything, bobID, room1_ID).Return(nil)
		rRepo.EXPECT().GetRoomType(mock.Anything, room1_ID).Return("group", nil)

		uRepo.EXPECT().GetInviteAvailability(mock.Anything, bobID).Return(true, nil)

		connectUser(t, h, uRepo, rRepo, bobID, bob, []*domain.Room{})

		dbRoom, err := h.CreateRoom(t.Context(), room1, aliceID)
		require.NoError(t, err)

		err = h.InviteUser(t.Context(), aliceID, bobID, dbRoom.ID)
		require.NoError(t, err)
	})
}

func Test_Hub_LeaveRoom(t *testing.T) {
	t.Run("success user offline", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, aliceID, room1_ID).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, room1_ID).Return(false, nil)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
		require.NoError(t, err)
	})

	t.Run("deletes room if empty", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)

		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, aliceID, room1_ID).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, room1_ID).Return(true, nil)
		rRepo.EXPECT().DeleteRoom(mock.Anything, room1_ID).Return((*domain.Room)(nil), nil)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
		require.NoError(t, err)
	})

	t.Run("success user online room removed when empty", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, aliceID, alice, []*domain.Room{{ID: room1_ID}})

		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, aliceID, room1_ID).Return(nil)
		rRepo.EXPECT().IsEmpty(mock.Anything, room1_ID).Return(false, nil)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
		require.NoError(t, err)
	})

	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(false, nil)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(false, errDB)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
		assert.Error(t, err)
	})

	t.Run("remove user repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		rRepo.EXPECT().RemoveUser(mock.Anything, aliceID, room1_ID).Return(errDB)

		err := h.LeaveRoom(t.Context(), aliceID, room1_ID)
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
		rRepo.EXPECT().CreateRoom(mock.Anything, room1, aliceID).Return(&domain.Room{ID: room1_ID, Name: &room1}, nil)

		room, err := h.CreateRoom(t.Context(), room1, aliceID)
		require.NoError(t, err)
		assert.Equal(t, room1_ID, room.ID)
		assert.Equal(t, room1, *room.Name)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().CreateRoom(mock.Anything, room1, aliceID).Return((*domain.Room)(nil), errDB)

		room, err := h.CreateRoom(t.Context(), room1, aliceID)
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}

func Test_Hub_CreateDM(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		fClientMock.EXPECT().IsFriend(mock.Anything, bobID, aliceID).Return(true, nil)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().CreateDM(mock.Anything, bobID, aliceID).Return(&domain.Room{ID: room1_ID}, nil)

		room, err := h.CreateDM(t.Context(), bobID, aliceID)
		require.NoError(t, err)
		assert.Equal(t, room1_ID, room.ID)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		fClientMock.EXPECT().IsFriend(mock.Anything, bobID, aliceID).Return(true, nil)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().CreateDM(mock.Anything, bobID, aliceID).Return((*domain.Room)(nil), errDB)

		room, err := h.CreateDM(t.Context(), bobID, aliceID)
		assert.Error(t, err)
		assert.Nil(t, room)
	})
}
func Test_Hub_GetRoomHistory(t *testing.T) {
	t.Run("not member returns ErrForbidden", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(false, nil)

		msgs, err := h.GetRoomHistory(t.Context(), room1_ID, aliceID, time.Time{})
		assert.ErrorIs(t, err, domain.ErrUserForbidden)
		assert.Nil(t, msgs)
	})

	t.Run("isMember error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(false, errDB)

		msgs, err := h.GetRoomHistory(t.Context(), room1_ID, aliceID, time.Time{})
		assert.Error(t, err)
		assert.Nil(t, msgs)
	})

	t.Run("before zero uses current time", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		expected := []*domain.Message{{Message: "hello"}}
		mRepo.EXPECT().GetMessagesBefore(mock.Anything, room1_ID, mock.Anything).Return(expected, nil)

		msgs, err := h.GetRoomHistory(t.Context(), room1_ID, aliceID, time.Time{})
		require.NoError(t, err)
		assert.Equal(t, expected, msgs)
	})

	t.Run("before past is passed through", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().IsMember(mock.Anything, aliceID, room1_ID).Return(true, nil)
		past := time.Now().Add(-1 * time.Hour)
		expected := []*domain.Message{{Message: "old message"}}
		mRepo.EXPECT().GetMessagesBefore(mock.Anything, room1_ID, past).Return(expected, nil)

		msgs, err := h.GetRoomHistory(t.Context(), room1_ID, aliceID, past)
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
		expected := []*domain.Room{{ID: room1_ID}, {ID: room2_ID}}
		rRepo.EXPECT().GetRoomsByUserID(mock.Anything, aliceID).Return(expected, nil)
		rRepo.EXPECT().GetDMsByUserID(mock.Anything, aliceID).Return([]*domain.Room{}, nil)

		rooms, err := h.GetRoomsByUser(t.Context(), aliceID)
		require.NoError(t, err)
		assert.Equal(t, expected, rooms)
	})

	t.Run("repo error", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		rRepo.EXPECT().GetRoomsByUserID(mock.Anything, aliceID).Return(([]*domain.Room)(nil), errDB)

		rooms, err := h.GetRoomsByUser(t.Context(), aliceID)
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

		assert.NotPanics(t, func() { h.Shutdown() })
	})

	t.Run("stops connected users and waits", func(t *testing.T) {
		uRepo := mocks.NewMockUserRepository(t)
		rRepo := mocks.NewMockRoomRepository(t)
		mRepo := mocks.NewMockMessageRepository(t)
		fClientMock := mocks.NewMockFriendshipClient(t)

		h := newHub(t, uRepo, rRepo, mRepo, fClientMock)
		connectUser(t, h, uRepo, rRepo, aliceID, alice, []*domain.Room{})

		done := make(chan struct{})
		go func() {
			h.Shutdown()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(3 * time.Second):
			t.Fatal("Shutdown did not complete in time")
		}
	})
}
