package domain

import "errors"

var (
	ErrUserInvalidInvitee     = errors.New("invalid invitee id")
	ErrFriendReqNotFound      = errors.New("friend request not found")
	ErrFriendNotFound         = errors.New("friend not found")
	ErrFriendReqAlreadyExists = errors.New("friend request already exists")
)
