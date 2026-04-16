package service

import (
	"context"
	"errors"
	"friends_service/internal/domain"
	"friends_service/internal/repository"
	"log/slog"
	"testing"

	"github.com/go-jose/go-jose/v4/testutils/assert"
	"github.com/go-jose/go-jose/v4/testutils/require"
	"github.com/stretchr/testify/mock"
)

type friendshipRepoMock struct{ mock.Mock }
type userRepoMock struct{ mock.Mock }

func (fr *friendshipRepoMock) GetFriends(ctx context.Context, userID string) ([]*domain.User, error) {
	args := fr.Called(ctx, userID)
	return args.Get(0).([]*domain.User), args.Error(1)
}

func (fr *friendshipRepoMock) AddFriend(ctx context.Context, inviterID, inviteeID string) error {
	args := fr.Called(ctx, inviterID, inviteeID)
	return args.Error(0)
}

func (fr *friendshipRepoMock) AcceptFriend(ctx context.Context, userID, inviterID string) (bool, error) {
	args := fr.Called(ctx, userID, inviterID)
	return args.Get(0).(bool), args.Error(1)
}

func (fr *friendshipRepoMock) DeclineFriend(ctx context.Context, userID, inviterID string) (bool, error) {
	args := fr.Called(ctx, userID, inviterID)
	return args.Get(0).(bool), args.Error(1)
}

func (fr *friendshipRepoMock) CancelFriend(ctx context.Context, userID, inviterID string) (bool, error) {
	args := fr.Called(ctx, userID, inviterID)
	return args.Get(0).(bool), args.Error(1)
}

func (fr *friendshipRepoMock) RemoveFriend(ctx context.Context, userID, friendID string) (bool, error) {
	args := fr.Called(ctx, userID, friendID)
	return args.Get(0).(bool), args.Error(1)
}

func (ur *userRepoMock) UserExists(ctx context.Context, userID string) (bool, error) {
	args := ur.Called(ctx, userID)
	return args.Get(0).(bool), args.Error(1)
}

func (ur *userRepoMock) GetUsersByUsername(ctx context.Context, name, cursor string) ([]*domain.User, error) {
	args := ur.Called(ctx, name, cursor)
	return args.Get(0).([]*domain.User), args.Error(1)
}

var _ repository.FriendshipRepository = (*friendshipRepoMock)(nil)
var _ repository.UserRepository = (*userRepoMock)(nil)

func setupSvc(t *testing.T, uRepo *userRepoMock, fRepo *friendshipRepoMock) Friendship {
	t.Helper()
	svc := NewFriendshipService(uRepo, fRepo, slog.Default())

	return svc
}

func Test_SendFriendRequest(t *testing.T) {
	alice := "alice"
	bob := "bob"
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		inviter string
		invitee string
		setup   func(ur *userRepoMock, fr *friendshipRepoMock)
		wantErr error
	}{
		{
			name:    "success",
			inviter: alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				ur.On("UserExists", mock.Anything, bob).Return(true, nil)
				fr.On("AddFriend", mock.Anything, alice, bob).Return(nil)
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
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				ur.On("UserExists", mock.Anything, bob).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:    "user does not exists",
			inviter: alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				ur.On("UserExists", mock.Anything, bob).Return(false, nil)
			},
			wantErr: domain.ErrUserInvalidInvitee,
		},
		{
			name:    "db error 2",
			inviter: alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				ur.On("UserExists", mock.Anything, bob).Return(true, nil)
				fr.On("AddFriend", mock.Anything, alice, bob).Return(errDB)
			},
			wantErr: errDB,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			fRepo := &friendshipRepoMock{}

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

			uRepo.AssertExpectations(t)
			fRepo.AssertExpectations(t)
		})
	}
}

func Test_AcceptFriendRequest(t *testing.T) {
	alice := "alice"
	bob := "bob"
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		user    string
		inviter string
		setup   func(ur *userRepoMock, fr *friendshipRepoMock)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("AcceptFriend", mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("AcceptFriend", mock.Anything, alice, bob).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:    "friend request not found",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("AcceptFriend", mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			fRepo := &friendshipRepoMock{}

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

			uRepo.AssertExpectations(t)
			fRepo.AssertExpectations(t)
		})
	}
}

func Test_DeclineFriendRequest(t *testing.T) {
	alice := "alice"
	bob := "bob"
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		user    string
		inviter string
		setup   func(ur *userRepoMock, fr *friendshipRepoMock)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("DeclineFriend", mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("DeclineFriend", mock.Anything, alice, bob).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:    "friend request not found",
			user:    alice,
			inviter: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("DeclineFriend", mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			fRepo := &friendshipRepoMock{}

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

			uRepo.AssertExpectations(t)
			fRepo.AssertExpectations(t)
		})
	}
}

func Test_CancelFriendRequest(t *testing.T) {
	alice := "alice"
	bob := "bob"
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		user    string
		invitee string
		setup   func(ur *userRepoMock, fr *friendshipRepoMock)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("CancelFriend", mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("CancelFriend", mock.Anything, alice, bob).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:    "friend request not found",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("CancelFriend", mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendReqNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			fRepo := &friendshipRepoMock{}

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

			uRepo.AssertExpectations(t)
			fRepo.AssertExpectations(t)
		})
	}
}

func Test_RemoveFriend(t *testing.T) {
	alice := "alice"
	bob := "bob"
	errDB := errors.New("db down")

	tests := []struct {
		name    string
		user    string
		invitee string
		setup   func(ur *userRepoMock, fr *friendshipRepoMock)
		wantErr error
	}{
		{
			name:    "success",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("RemoveFriend", mock.Anything, alice, bob).Return(true, nil)
			},
		},
		{
			name:    "db error",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("RemoveFriend", mock.Anything, alice, bob).Return(false, errDB)
			},
			wantErr: errDB,
		},
		{
			name:    "friend request not found",
			user:    alice,
			invitee: bob,
			setup: func(ur *userRepoMock, fr *friendshipRepoMock) {
				fr.On("RemoveFriend", mock.Anything, alice, bob).Return(false, nil)
			},
			wantErr: domain.ErrFriendNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uRepo := &userRepoMock{}
			fRepo := &friendshipRepoMock{}

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

			uRepo.AssertExpectations(t)
			fRepo.AssertExpectations(t)
		})
	}
}
