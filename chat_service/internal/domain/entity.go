package domain

import "time"

type Room struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedBy string    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Message struct {
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	RoomID    string    `json:"roomId"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
