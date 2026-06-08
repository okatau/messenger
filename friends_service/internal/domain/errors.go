package domain

import "errors"

var (
	ErrRequestAlreadyExists = errors.New("request already exists")
	ErrUserNotFound         = errors.New("user not found")
	ErrRequestNotFound      = errors.New("request not found")
)
