package repository

import (
	"context"
	"errors"
	"friends_service/internal/domain"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const pgUniqueViolation = "23505"

type FriendshipRepository interface {
	GetFriends(ctx context.Context, userID string) ([]*domain.User, error)
	AddFriend(ctx context.Context, inviterID, inviteeID string) error
	AcceptFriend(ctx context.Context, userID, inviterID string) (bool, error)
	DeclineFriend(ctx context.Context, userID, inviterID string) (bool, error)
	CancelFriend(ctx context.Context, userID, inviterID string) (bool, error)
	RemoveFriend(ctx context.Context, userID, friendID string) (bool, error)
}

type friendshipRepo struct {
	pool *pgxpool.Pool
}

func NewFriendshipRepository(pool *pgxpool.Pool) FriendshipRepository {
	return &friendshipRepo{pool: pool}
}

func (r *friendshipRepo) GetFriends(ctx context.Context, userID string) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.created_at
		FROM friendships f
		JOIN users u ON u.id = f.requester_id
		WHERE f.addressee_id = $1 AND f.status = 'accepted'
		
		UNION
		
		SELECT u.id, u.name, u.email, u.created_at
		FROM friendships f
		JOIN users u ON u.id = f.addressee_id
		WHERE f.requester_id = $1 AND f.status = 'accepted'
	`

	var friends []*domain.User

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var f domain.User
		if err := rows.Scan(&f.ID, &f.Username, &f.Email, &f.CreatedAt); err != nil {
			return nil, err
		}
		friends = append(friends, &f)
	}

	return friends, rows.Err()
}

func (r *friendshipRepo) AddFriend(ctx context.Context, inviterID, inviteeID string) error {
	query := `
		INSERT INTO friendships (requester_id, addressee_id)
		VALUES ($1, $2)
	`

	_, err := r.pool.Exec(ctx, query, inviterID, inviteeID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation { // unique_violation
			return domain.ErrFriendReqAlreadyExists
		}
		return err
	}
	return nil
}

func (r *friendshipRepo) AcceptFriend(ctx context.Context, userID, inviterID string) (bool, error) {
	query := `
		UPDATE friendships
		SET status = 'accepted', updated_at = now()
		WHERE 
			addressee_id = $1 AND 
			requester_id = $2 AND 
			status = 'pending'
	`

	tag, err := r.pool.Exec(ctx, query, userID, inviterID)
	return tag.RowsAffected() > 0, err
}

func (r *friendshipRepo) DeclineFriend(ctx context.Context, userID, inviterID string) (bool, error) {
	query := `
		UPDATE friendships
		SET status = 'declined', updated_at = now()
		WHERE 
			addressee_id = $1 AND 
			requester_id = $2 AND
			status = 'pending'
	`

	tag, err := r.pool.Exec(ctx, query, userID, inviterID)
	return tag.RowsAffected() > 0, err
}

func (r *friendshipRepo) CancelFriend(ctx context.Context, userID, inviteeID string) (bool, error) {
	query := `
		UPDATE friendships
		SET status = 'cancelled', updated_at = now()
		WHERE 
			requester_id = $1 AND 
			addressee_id = $2 AND
			status = 'pending'
	`

	tag, err := r.pool.Exec(ctx, query, userID, inviteeID)
	return tag.RowsAffected() > 0, err
}

func (r *friendshipRepo) RemoveFriend(ctx context.Context, userID, friendID string) (bool, error) {
	query := `
		DELETE FROM friendships
		WHERE 
			((addressee_id = $1 AND requester_id = $2) OR 
			(addressee_id = $2 AND requester_id = $1)) AND
			status = 'accepted'
	`

	tag, err := r.pool.Exec(ctx, query, userID, friendID)
	return tag.RowsAffected() > 0, err
}
