package repository

import (
	"context"

	"auth_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository interface {
	GetUserByID(ctx context.Context, id string) (*domain.User, error)
	GetUserByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, name, email, passwordHash string) (*domain.User, error)
	DeleteUser(ctx context.Context, id string) (*domain.User, error)
}

type userRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepo{pool: pool}
}

func (r *userRepo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *userRepo) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, name, email, password_hash, created_at
		FROM users
		WHERE email = $1
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &user, err
}

func (r *userRepo) CreateUser(ctx context.Context, name, email, passwordHash string) (*domain.User, error) {
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, name, email, password_hash, created_at
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, name, email, passwordHash).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return &user, err
}

func (r *userRepo) DeleteUser(ctx context.Context, id string) (*domain.User, error) {
	query := `
		DELETE FROM users
		WHERE id = $1
		RETURNING id, name, email, password_hash, created_at
	`

	var user domain.User
	err := r.pool.QueryRow(ctx, query, id).Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.CreatedAt)
	return &user, err
}
