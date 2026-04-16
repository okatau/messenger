package repository

import (
	"auth_service/internal/domain"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	AddUser(ctx context.Context, name, email, passwordHash string) (*domain.User, error)
	RemoveUser(ctx context.Context, id string) (*domain.User, error)
}

type userRepoPG struct {
	pool *pgxpool.Pool
}

func NewUserRepositoryPG(pool *pgxpool.Pool) UserRepository {
	return &userRepoPG{pool: pool}
}

func (r *userRepoPG) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *userRepoPG) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *userRepoPG) AddUser(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash, created_at
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, name, email, passwordHash).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return &user, err
}

func (r *userRepoPG) RemoveUser(ctx context.Context, id string) (*domain.User, error) {
	query := `
		DELETE FROM users
		WHERE id = $1
		RETURNING id, name, email, password_hash, created_at
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return &user, err
}
