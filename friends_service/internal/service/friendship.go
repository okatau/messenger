package service

import (
	"context"
	"log/slog"

	"friends_service/internal/domain"
	"friends_service/internal/repository"
	sl "friends_service/pkg/service_logger"
)

const svcName = "service.friendship"

type Friendship interface {
	SendFriendRequest(ctx context.Context, inviterID, inviteeID string) error
	AcceptFriendRequest(ctx context.Context, userID, inviterID string) error
	DeclineFriendRequest(ctx context.Context, userID, inviterID string) error
	CancelFriendRequest(ctx context.Context, userID, inviteeID string) error
	RemoveFriend(ctx context.Context, userID, friendID string) error

	GetFriendsList(ctx context.Context, userID string) ([]*domain.User, error)
	GetInvites(ctx context.Context, userID string) ([]*domain.User, error)
	IsFriend(ctx context.Context, userID, friendID string) (bool, error)

	SearchFriends(ctx context.Context, userID, searchUsername, cursor string) ([]*domain.User, error)
	SearchUsers(ctx context.Context, username, cursor string) ([]*domain.User, error)
}

type friendship struct {
	userRepo       repository.UserRepository
	friendshipRepo repository.FriendshipRepository
	logger         *slog.Logger
}

func NewFriendshipService(
	userRepo repository.UserRepository,
	friendshipRepo repository.FriendshipRepository,
	logger *slog.Logger,
) Friendship {
	return &friendship{
		userRepo:       userRepo,
		friendshipRepo: friendshipRepo,
		logger:         logger,
	}
}

func (f *friendship) SendFriendRequest(ctx context.Context, inviterID, inviteeID string) error {
	l := f.loggerWith(".sendfriendrequest")

	inviteeExists, err := f.userRepo.UserExists(ctx, inviteeID)
	if err != nil {
		l.Error("failed to get invitee existence", sl.Err(err))
		return err
	}
	if !inviteeExists {
		return domain.ErrUserNotFound
	}

	return f.friendshipRepo.AddFriend(ctx, inviterID, inviteeID)
}

func (f *friendship) AcceptFriendRequest(ctx context.Context, userID, inviterID string) error {
	l := f.loggerWith(".acceptfriendrequest")

	accepted, err := f.friendshipRepo.AcceptFriend(ctx, userID, inviterID)
	if err != nil {
		l.Error("fauled to accept friendship request", sl.Err(err))
		return err
	}
	if !accepted {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (f *friendship) DeclineFriendRequest(ctx context.Context, userID, inviterID string) error {
	l := f.loggerWith(".declinefriendrequest")

	declined, err := f.friendshipRepo.DeclineFriend(ctx, userID, inviterID)
	if err != nil {
		l.Error("failed to decline friendship request", sl.Err(err))
		return err
	}
	if !declined {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (f *friendship) CancelFriendRequest(ctx context.Context, userID, inviteeID string) error {
	l := f.loggerWith(".cancelfriendrequest")

	canceled, err := f.friendshipRepo.CancelFriend(ctx, userID, inviteeID)
	if err != nil {
		l.Error("failed to cancel friendship request", sl.Err(err))
		return err
	}
	if !canceled {
		return domain.ErrRequestNotFound
	}
	return nil
}

func (f *friendship) RemoveFriend(ctx context.Context, userID, friendID string) error {
	l := f.loggerWith(".removefriend")

	removed, err := f.friendshipRepo.RemoveFriend(ctx, userID, friendID)
	if err != nil {
		l.Error("failed to remove friend", sl.Err(err))
		return err
	}
	if !removed {
		return domain.ErrUserNotFound
	}
	return nil
}

func (f *friendship) GetFriendsList(ctx context.Context, userID string) ([]*domain.User, error) {
	l := f.loggerWith(".getfriendslist")

	friends, err := f.friendshipRepo.GetFriends(ctx, userID)
	if err != nil {
		l.Error("failed to get friends", sl.Err(err))
		return nil, err
	}
	return friends, err
}

func (f *friendship) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	return f.friendshipRepo.IsFriend(ctx, userID, friendID)
}

func (f *friendship) GetInvites(ctx context.Context, userID string) ([]*domain.User, error) {
	l := f.loggerWith(".getinvites")

	invites, err := f.friendshipRepo.GetInvites(ctx, userID)
	if err != nil {
		l.Error("failed to get invites", sl.Err(err))
		return nil, err
	}
	return invites, nil
}

func (f *friendship) SearchUsers(ctx context.Context, username, cursor string) ([]*domain.User, error) {
	l := f.loggerWith(".searchuser")

	users, err := f.userRepo.GetUsersByUsername(ctx, username, cursor)
	if err != nil {
		l.Error("failed to get users", sl.Err(err))
		return nil, err
	}
	return users, err
}

func (f *friendship) SearchFriends(ctx context.Context, userID, searchUsername, cursor string) ([]*domain.User, error) {
	l := f.loggerWith(".searchfriend")

	invites, err := f.friendshipRepo.GetFriendByUsername(ctx, userID, searchUsername, cursor)
	if err != nil {
		l.Error("failed to search friend", sl.Err(err))
		return nil, err
	}
	return invites, nil
}

func (f *friendship) loggerWith(fnName string) *slog.Logger {
	return f.logger.With("op", svcName+fnName)
}
