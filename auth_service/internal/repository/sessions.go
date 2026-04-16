package repository

import (
	"auth_service/internal/domain"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository interface {
	GetSessionByToken(ctx context.Context, refreshToken string) (*domain.Session, error)
	AddSession(ctx context.Context, userID, name, refreshToken string, expiresAt time.Time) error
	RemoveSession(ctx context.Context, refreshToken string) (*domain.Session, error)
	RemoveSessionsByUserID(ctx context.Context, userID string) ([]*domain.Session, error)
}

type sessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepositoryPG(pool *pgxpool.Pool) SessionRepository {
	return &sessionRepo{pool: pool}
}

func (r *sessionRepo) GetSessionByToken(ctx context.Context, refreshToken string) (*domain.Session, error) {
	query := `
		SELECT id, user_id, name, refresh_token, expires_at
		FROM sessions
		WHERE refresh_token = $1
	`
	var session domain.Session
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(&session.ID, &session.UserID, &session.Username, &session.RefreshToken, &session.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &session, err
}

func (r *sessionRepo) AddSession(ctx context.Context, userID, name, refreshToken string, expiresAt time.Time) error {
	query := `
		INSERT INTO sessions (user_id, name, refresh_token, expires_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := r.pool.Exec(ctx, query, userID, name, refreshToken, expiresAt)
	return err
}

func (r *sessionRepo) RemoveSession(ctx context.Context, refreshToken string) (*domain.Session, error) {
	query := `
		DELETE FROM sessions
		WHERE refresh_token = $1
		RETURNING id, user_id, name, refresh_token, expires_at
	`
	var session domain.Session
	err := r.pool.QueryRow(ctx, query, refreshToken).Scan(&session.ID, &session.UserID, &session.Username, &session.RefreshToken, &session.ExpiresAt)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	return &session, err
}

func (r *sessionRepo) RemoveSessionsByUserID(ctx context.Context, userID string) ([]*domain.Session, error) {
	query := `
		DELETE FROM sessions
		WHERE user_id = $1
		RETURNING id, user_id, name, refresh_token, expires_at
	`

	var sessions []*domain.Session
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var s domain.Session
		if err := rows.Scan(
			&s.ID,
			&s.UserID,
			&s.Username,
			&s.RefreshToken,
			&s.ExpiresAt,
		); err != nil {
			return nil, err
		}

		sessions = append(sessions, &s)
	}
	return sessions, rows.Err()

}
