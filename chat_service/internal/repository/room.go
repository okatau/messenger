package repository

import (
	"context"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepository interface {
	CreateDM(ctx context.Context, user1ID, user2ID string) (*domain.Room, error)
	CreateRoom(ctx context.Context, name, userID string) (*domain.Room, error)
	DeleteRoom(ctx context.Context, roomID string) (*domain.Room, error)
	AddUser(ctx context.Context, userID, roomID string) error
	RemoveUser(ctx context.Context, userID, roomID string) error

	IsMember(ctx context.Context, userID, roomID string) (bool, error)
	IsEmpty(ctx context.Context, roomID string) (bool, error)

	GetAllRooms(ctx context.Context) ([]*domain.Room, error)
	GetRoomsByUserID(ctx context.Context, userID string) ([]*domain.Room, error)
	GetUsersByRoomID(ctx context.Context, roomID string) ([]*domain.User, error)
	GetDMsByUserID(ctx context.Context, userID string) ([]*domain.Room, error)
	GetRoomType(ctx context.Context, roomID string) (string, error)
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
		SELECT r.id, r.name, r.created_by, r.created_at 
		FROM rooms r
		JOIN room_members rm ON rm.room_id = r.id
		WHERE rm.user_id = $1 AND r.type = 'group'
	`

	var rooms []*domain.Room
	rows, err := r.pool.Query(ctx, query, userID)
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
	defer tx.Rollback(ctx) //nolint:errcheck // rollback is called as cleanup; if tx was committed, rollback returns harmless ErrTxClosed

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

func (r *roomRepo) CreateDM(ctx context.Context, user1ID, user2ID string) (*domain.Room, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}

	defer tx.Rollback(ctx) //nolint:errcheck // ignores if tx was successful

	query := `
		INSERT INTO rooms(created_by, type)
		VALUES ($1, 'direct')
		RETURNING id, created_by, created_at
	`
	var room domain.Room
	if err := tx.QueryRow(ctx, query, user1ID).Scan(&room.ID, &room.CreatedBy, &room.CreatedAt); err != nil { //nolint:govet // does not matter
		return nil, err
	}

	query = `
		INSERT INTO direct_conversations(room_id, user1_id, user2_id)
		VALUES ($1, LEAST($2::uuid, $3::uuid), GREATEST($2::uuid, $3::uuid))
	`
	if _, err := tx.Exec(ctx, query, room.ID, user1ID, user2ID); err != nil { //nolint:govet // does not matter
		return nil, err
	}

	query = `
		INSERT INTO room_members(room_id, user_id)
		VALUES 
			($1, $2),
			($1, $3)
	`
	if _, err := tx.Exec(ctx, query, room.ID, user1ID, user2ID); err != nil { //nolint:govet // does not matter
		return nil, err
	}

	err = tx.Commit(ctx)
	return &room, err
}

func (r *roomRepo) GetDMsByUserID(ctx context.Context, userID string) ([]*domain.Room, error) {
	query := `
		SELECT
			r.id          AS room_id,
			CASE
			WHEN dv.user1_id = $1 THEN u2.name
			ELSE u1.name
			END AS other_user_name,
			r.created_by,
			r.created_at
		FROM direct_conversations dv
		JOIN rooms r  ON r.id  = dv.room_id
		JOIN users u1 ON u1.id = dv.user1_id
		JOIN users u2 ON u2.id = dv.user2_id
		WHERE dv.user1_id = $1
			OR dv.user2_id = $1
	`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rooms []*domain.Room
	for rows.Next() {
		var r domain.Room

		if err := rows.Scan(&r.ID, &r.Name, &r.CreatedBy, &r.CreatedAt); err != nil {
			return nil, err
		}

		rooms = append(rooms, &r)
	}

	return rooms, rows.Err()
}

func (r *roomRepo) GetRoomType(ctx context.Context, roomID string) (string, error) {
	query := `
		SELECT type
		FROM rooms
		WHERE id = $1
	`
	var roomType string
	err := r.pool.QueryRow(ctx, query, roomID).Scan(&roomType)
	return roomType, err
}
