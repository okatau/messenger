package service

import (
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/mocks"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var (
	alice   = "alice"
	bob     = "bob"
	dbError = errors.New("db down")
)

func setupSvc(t *testing.T, uRepo *mocks.MockUserRepository, fRepo *mocks.MockFriendshipRepository) Friendship {
	t.Helper()
	svc := NewFriendshipService(uRepo, fRepo, slog.Default())

	return svc
}

func Test_SendFriendRequest(t *testing.T) {

	tests := []struct {
		name    string
		inviter string
		invitee string
		setup   func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository)
		wantErr error
	}{
		{
			name:    "success",
			inviter: alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				ur.EXPECT().UserExists(mock.Anything, bob).Return(true, nil)
				fr.EXPECT().AddFriend(mock.Anything, alice, bob).Return(nil)
			},
		},
		{
			name:    "invite herself",
			inviter: alice,
			invitee: alice,
			wantErr: domain.ErrUserInvalidInvitee,
		},
		{
			name:    "db error 1",
			inviter: alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				ur.EXPECT().UserExists(mock.Anything, bob).Return(false, dbError)
			},
			wantErr: dbError,
		},
		{
			name:    "user does not exists",
			inviter: alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				ur.EXPECT().UserExists(mock.Anything, bob).Return(false, nil)
			},
			wantErr: domain.ErrUserInvalidInvitee,
		},
		{
			name:    "db error 2",
			inviter: alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				ur.EXPECT().UserExists(mock.Anything, bob).Return(true, nil)
				fr.EXPECT().AddFriend(mock.Anything, alice, bob).Return(dbError)
			},
			wantErr: dbError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			fRepo := mocks.NewMockFriendshipRepository(t)

			if tt.setup != nil {
				tt.setup(uRepo, fRepo)
			}

			svc := setupSvc(t, uRepo, fRepo)

			err := svc.SendFriendRequest(t.Context(), tt.inviter, tt.invitee)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_AcceptFriendRequest(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		inviter string
		setup   func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().AcceptFriend(mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().AcceptFriend(mock.Anything, alice, bob).Return(false, dbError)
			},
			wantErr: dbError,
		},
		{
			name:    "friend request not found",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().AcceptFriend(mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			fRepo := mocks.NewMockFriendshipRepository(t)

			if tt.setup != nil {
				tt.setup(uRepo, fRepo)
			}

			svc := setupSvc(t, uRepo, fRepo)

			err := svc.AcceptFriendRequest(t.Context(), tt.user, tt.inviter)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_DeclineFriendRequest(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		inviter string
		setup   func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().DeclineFriend(mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().DeclineFriend(mock.Anything, alice, bob).Return(false, dbError)
			},
			wantErr: dbError,
		},
		{
			name:    "friend request not found",
			user:    alice,
			inviter: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().DeclineFriend(mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			fRepo := mocks.NewMockFriendshipRepository(t)

			if tt.setup != nil {
				tt.setup(uRepo, fRepo)
			}

			svc := setupSvc(t, uRepo, fRepo)

			err := svc.DeclineFriendRequest(t.Context(), tt.user, tt.inviter)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_CancelFriendRequest(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		invitee string
		setup   func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().CancelFriend(mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().CancelFriend(mock.Anything, alice, bob).Return(false, dbError)
			},
			wantErr: dbError,
		},
		{
			name:    "friend request not found",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().CancelFriend(mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			fRepo := mocks.NewMockFriendshipRepository(t)

			if tt.setup != nil {
				tt.setup(uRepo, fRepo)
			}

			svc := setupSvc(t, uRepo, fRepo)

			err := svc.CancelFriendRequest(t.Context(), tt.user, tt.invitee)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_RemoveFriend(t *testing.T) {
	tests := []struct {
		name    string
		user    string
		invitee string
		setup   func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().RemoveFriend(mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().RemoveFriend(mock.Anything, alice, bob).Return(false, dbError)
			},
			wantErr: dbError,
		},
		{
			name:    "friend request not found",
			user:    alice,
			invitee: bob,
			setup: func(ur *mocks.MockUserRepository, fr *mocks.MockFriendshipRepository) {
				fr.EXPECT().RemoveFriend(mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := mocks.NewMockUserRepository(t)
			fRepo := mocks.NewMockFriendshipRepository(t)

			if tt.setup != nil {
				tt.setup(uRepo, fRepo)
			}

			svc := setupSvc(t, uRepo, fRepo)

			err := svc.RemoveFriend(t.Context(), tt.user, tt.invitee)

			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
