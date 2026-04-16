package repository

import (
	"context"
	"friends_service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

const pageSize = 10

type UserRepository interface {
	UserExists(ctx context.Context, userID string) (bool, error)
	GetUsersByUsername(ctx context.Context, name, cursor string) ([]*domain.User, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) UserExists(ctx context.Context, userID string) (bool, error) {
	query := `
		SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)
	`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	return exists, err
}

func (r *userRepo) GetUsersByUsername(ctx context.Context, name, cursor string) ([]*domain.User, error) {
	query := `
		SELECT id, email, name, created_at
		FROM users
		WHERE 
			name ILIKE $1 || '%' AND 
			name > $2
		ORDER BY name
		LIMIT $3
	`

	var users []*domain.User

	rows, err := r.pool.Query(ctx, query, name, cursor, pageSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var u domain.User
		if err := rows.Scan(
			&u.ID,
			&u.Email,
			&u.Name,
			&u.CreatedAt,
		); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
