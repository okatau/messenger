package repository

import (
	"chat_service/internal/domain"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, name, email
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Name, &user.Email)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}
