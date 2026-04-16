package domain

import "time"

type User struct {
	ID           string    `json:"id"`
	Name         string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"createdAt"`
}
