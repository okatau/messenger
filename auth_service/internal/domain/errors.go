package domain

import "errors"

var (
	ErrUserExist     = errors.New("user already exist")
	ErrUserNotFound  = errors.New("user not found")
	ErrUserForbidden = errors.New("user forbidden")
	ErrTokenNotFound = errors.New("token not found")
	ErrTokenExpired  = errors.New("token expired")
)
