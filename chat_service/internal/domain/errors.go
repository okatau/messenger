package domain

import "errors"

var (
	ErrUserNotFound     = errors.New("user not found")
	ErrUserExists       = errors.New("user exists")
	ErrUserNoRooms      = errors.New("user doesnt have chats")
	ErrUserDisconnected = errors.New("user disconnected")
	ErrRoomExists       = errors.New("room exists")
	ErrRoomNotFound     = errors.New("room not found")
	ErrForbidden        = errors.New("forbidden")
)
