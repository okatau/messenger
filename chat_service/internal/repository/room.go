package repository

import (
	"context"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepository interface {
	GetAllRooms(ctx context.Context) ([]*domain.Room, error)
	GetRoomsByUserID(ctx context.Context, userID string) ([]*domain.Room, error)
	GetUsersByRoomID(ctx context.Context, roomID string) ([]*domain.User, error)
	CreateRoom(ctx context.Context, name, userID string) (*domain.Room, error)
	DeleteRoom(ctx context.Context, roomID string) (*domain.Room, error)
	AddUser(ctx context.Context, userID, roomID string) error
	RemoveUser(ctx context.Context, userID, roomID string) error
	IsMember(ctx context.Context, userID, roomID string) (bool, error)
	IsEmpty(ctx context.Context, roomID string) (bool, error)
}

type roomRepo struct {
	pool *pgxpool.Pool
}

func NewRoomRepository(pool *pgxpool.Pool) RoomRepository {
	return &roomRepo{pool: pool}
}

func (r *roomRepo) GetAllRooms(ctx context.Context) ([]*domain.Room, error) {
	query := `
		SELECT id, name, created_by, created_at
		FROM rooms
	`

	var rooms []*domain.Room
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var room domain.Room
		if err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, rows.Err()
}

func (r *roomRepo) GetRoomsByUserID(ctx context.Context, userID string) ([]*domain.Room, error) {
	query := `
		SELECT r.id, r.name, r.created_by, r.created_at FROM rooms r
		JOIN room_members rm ON rm.room_id = r.id
		WHERE rm.user_id = $1
	`

	var rooms []*domain.Room
	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var room domain.Room
		if err := rows.Scan(
			&room.ID,
			&room.Name,
			&room.CreatedBy,
			&room.CreatedAt,
		); err != nil {
			return nil, err
		}
		rooms = append(rooms, &room)
	}

	return rooms, rows.Err()
}

func (r *roomRepo) GetUsersByRoomID(ctx context.Context, roomID string) ([]*domain.User, error) {
	query := `
		SELECT u.id, u.name, u.created_at FROM users u
		JOIN room_members rm ON rm.user_id = u.id
		WHERE rm.room_id = $1
	`

	var users []*domain.User

	rows, err := r.pool.Query(ctx, query, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Username, &user.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}
	return users, rows.Err()
}

func (r *roomRepo) CreateRoom(ctx context.Context, name, userID string) (*domain.Room, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var room domain.Room
	err = tx.QueryRow(ctx, `
		INSERT INTO rooms(name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at
	`, name, userID).Scan(&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt)
	if err != nil {
		return nil, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO room_members (user_id, room_id)
		VALUES ($1, $2)
	`, userID, room.ID)
	if err != nil {
		return nil, err
	}

	return &room, tx.Commit(ctx)
}

func (r *roomRepo) DeleteRoom(ctx context.Context, roomID string) (*domain.Room, error) {
	query := `
		DELETE FROM rooms
		WHERE id = $1
		RETURNING id, name, created_by, created_at
	`

	var room domain.Room
	err := r.pool.QueryRow(ctx, query, roomID).Scan(&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt)
	return &room, err
}

func (r *roomRepo) AddUser(ctx context.Context, userID, roomID string) error {
	query := `
		INSERT INTO room_members (user_id, room_id)
		VALUES ($1, $2)
	`

	_, err := r.pool.Exec(ctx, query, userID, roomID)
	return err
}

func (r *roomRepo) RemoveUser(ctx context.Context, userID, roomID string) error {
	query := `
		DELETE FROM room_members
		WHERE user_id = $1 AND room_id = $2
	`

	_, err := r.pool.Exec(ctx, query, userID, roomID)
	return err
}

func (r *roomRepo) IsMember(ctx context.Context, userID, roomID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM room_members
			WHERE room_id = $1 AND user_id = $2
		)	
	`

	var isMember bool
	err := r.pool.QueryRow(ctx, query, roomID, userID).Scan(&isMember)
	return isMember, err
}

func (r *roomRepo) IsEmpty(ctx context.Context, roomID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM room_members
			WHERE room_id = $1
		)
	`

	var userExists bool
	err := r.pool.QueryRow(ctx, query, roomID).Scan(&userExists)
	return !userExists, err
}
