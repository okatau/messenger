package service

import (
	"context"
	"friends_service/internal/domain"
	"friends_service/internal/repository"
	"friends_service/pkg/service_logger"
	"log/slog"
)

type Friendship interface {
	SendFriendRequest(ctx context.Context, inviterID, inviteeID string) error
	AcceptFriendRequest(ctx context.Context, userID, inviterID string) error
	DeclineFriendRequest(ctx context.Context, userID, inviterID string) error
	CancelFriendRequest(ctx context.Context, userID, inviteeID string) error
	RemoveFriend(ctx context.Context, userID, friendID string) error

	GetFriendsList(ctx context.Context, userID string) ([]*domain.User, error)
	GetInvites(ctx context.Context, userID string) ([]*domain.User, error)
	IsFriend(ctx context.Context, userID, friendID string) (bool, error)

	SearchFriend(ctx context.Context, userID, searchUsername, cursor string) ([]*domain.User, error)
	SearchUser(ctx context.Context, username, cursor string) ([]*domain.User, error)
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
	const op = "service.friendship.sendfriendrequest"
	logger := f.logger.With(slog.String("op", op))

	if inviterID == inviteeID {
		return domain.ErrUserInvalidInvitee
	}
	inviteeExists, err := f.userRepo.UserExists(ctx, inviteeID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return err
	}
	if !inviteeExists {
		return domain.ErrUserInvalidInvitee
	}

	return f.friendshipRepo.AddFriend(ctx, inviterID, inviteeID)
}

func (f *friendship) AcceptFriendRequest(ctx context.Context, userID, inviterID string) error {
	const op = "service.friendship.acceptfriendrequest"
	logger := f.logger.With(slog.String("op", op))

	accepted, err := f.friendshipRepo.AcceptFriend(ctx, userID, inviterID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return err
	}
	if !accepted {
		return domain.ErrFriendReqNotFound
	}
	return nil
}

func (f *friendship) DeclineFriendRequest(ctx context.Context, userID, inviterID string) error {
	const op = "service.friendship.declinefriendrequest"
	logger := f.logger.With(slog.String("op", op))

	declined, err := f.friendshipRepo.DeclineFriend(ctx, userID, inviterID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return err
	}
	if !declined {
		return domain.ErrFriendReqNotFound
	}
	return nil
}

func (f *friendship) CancelFriendRequest(ctx context.Context, userID, inviteeID string) error {
	const op = "service.friendship.cancelfriendrequest"
	logger := f.logger.With(slog.String("op", op))

	cancelled, err := f.friendshipRepo.CancelFriend(ctx, userID, inviteeID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return err
	}
	if !cancelled {
		return domain.ErrFriendReqNotFound
	}
	return nil
}

func (f *friendship) RemoveFriend(ctx context.Context, userID, friendID string) error {
	const op = "service.friendship.removefriend"
	logger := f.logger.With(slog.String("op", op))

	removed, err := f.friendshipRepo.RemoveFriend(ctx, userID, friendID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return err
	}
	if !removed {
		return domain.ErrFriendNotFound
	}
	return nil
}

func (f *friendship) GetFriendsList(ctx context.Context, userID string) ([]*domain.User, error) {
	const op = "service.friendship.getfriendslist"
	logger := f.logger.With(slog.String("op", op))

	friends, err := f.friendshipRepo.GetFriends(ctx, userID)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return nil, err
	}
	return friends, err
}

func (f *friendship) IsFriend(ctx context.Context, userID, friendID string) (bool, error) {
	return f.friendshipRepo.IsFriend(ctx, userID, friendID)
}

func (f *friendship) GetInvites(ctx context.Context, userID string) ([]*domain.User, error) {
	const op = "service.friendship.getinvites"
	logger := f.logger.With(slog.String("op", op))

	invites, err := f.friendshipRepo.GetInvites(ctx, userID)
	if err != nil {
		logger.Error("error reading pending invites", service_logger.Err(err))
		return nil, err
	}
	return invites, nil
}

func (f *friendship) SearchUser(ctx context.Context, username, cursor string) ([]*domain.User, error) {
	const op = "service.friendship.searchuser"
	logger := f.logger.With(slog.String("op", op))

	users, err := f.userRepo.GetUsersByUsername(ctx, username, cursor)
	if err != nil {
		logger.Error("error reading db", service_logger.Err(err))
		return nil, err
	}
	return users, err
}

func (f *friendship) SearchFriend(ctx context.Context, userID, searchUsername, cursor string) ([]*domain.User, error) {
	const op = "service.friendship.searchfriend"
	logger := f.logger.With(slog.String("op", op))

	invites, err := f.friendshipRepo.GetFriendByUsername(ctx, userID, searchUsername, cursor)
	if err != nil {
		logger.Error("error reading pending invites", service_logger.Err(err))
		return nil, err
	}
	return invites, nil
}
