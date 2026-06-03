package repository

import (
	"context"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	ChangeInviteAvailability(ctx context.Context, userID string, availability bool) error

	GetUserByID(ctx context.Context, userID string) (*domain.User, error)
	GetInviteAvailability(ctx context.Context, userID string) (bool, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) GetUserByID(ctx context.Context, userID string) (*domain.User, error) {
	query := `
		SELECT id, name, email
		FROM users
		WHERE id = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, userID).Scan(&user.ID, &user.Username, &user.Email)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *userRepo) GetInviteAvailability(ctx context.Context, userID string) (bool, error) {
	query := `
		SELECT available 
		FROM invite_available 
		WHERE user_id = $1
	`

	var available bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&available)
	return available, err
}

func (r *userRepo) ChangeInviteAvailability(ctx context.Context, userID string, availability bool) error {
	query := `
		UPDATE invite_available
		SET available = $1
		WHERE user_id = $2
	`

	_, err := r.pool.Exec(ctx, query, availability, userID)
	return err
}
