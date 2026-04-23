package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// TODO посмотреть как берется сообщения при случае когда в кэше > 50 сообщений

const (
	cacheSize = 50
	cacheTTL  = 24 * time.Hour
)

type MessageRepository interface {
	WriteMessage(ctx context.Context, message *domain.Message) error
	GetMessages(ctx context.Context, roomID string) ([]*domain.Message, error)
	GetMessagesBefore(ctx context.Context, roomID string, before time.Time) ([]*domain.Message, error)
}

type messageRepo struct {
	pool *pgxpool.Pool
	rdb  redis.UniversalClient
}

func NewMessageRepository(pool *pgxpool.Pool, rdb redis.UniversalClient) MessageRepository {
	return &messageRepo{
		pool: pool,
		rdb:  rdb,
	}
}

func cacheKey(roomID string) string {
	return fmt.Sprintf("room:%s:messages", roomID)
}

func (r *messageRepo) WriteMessage(ctx context.Context, message *domain.Message) error {
	query := `
		INSERT INTO messages (room_id, sender_id, body, created_at)
		VALUES ($1, $2, $3, $4)
	`

	_, err := r.pool.Exec(ctx, query, message.RoomID, message.UserID, message.Message, message.Timestamp)
	if err != nil {
		return err
	}

	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	key := cacheKey(message.RoomID)

	pipe := r.rdb.Pipeline()
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(message.Timestamp.Unix()), Member: string(data)})
	pipe.ZRemRangeByRank(ctx, key, 0, -cacheSize-1)
	pipe.Expire(ctx, key, cacheTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (r *messageRepo) GetMessages(ctx context.Context, roomID string) ([]*domain.Message, error) {
	key := cacheKey(roomID)
	cached, err := r.rdb.ZRangeArgs(ctx, redis.ZRangeArgs{Key: key, Start: 0, Stop: cacheSize - 1, Rev: true}).Result()

	if err == nil && len(cached) > 0 {
		return deserializeMessage(cached)
	}

	query := `
		SELECT m.sender_id, u.name, m.room_id, m.body, m.created_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, roomID, cacheSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.UserID, &msg.Username, &msg.RoomID, &msg.Message, &msg.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	if len(messages) > 0 {
		r.warmCache(ctx, key, messages)
	}
	return messages, rows.Err()
}

func (r *messageRepo) GetMessagesBefore(ctx context.Context, roomID string, before time.Time) ([]*domain.Message, error) {
	query := `
		SELECT m.sender_id, u.name, m.room_id, m.body, m.created_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.room_id = $1 AND m.created_at < $2
		ORDER BY m.created_at DESC
		LIMIT $3
	`

	rows, err := r.pool.Query(ctx, query, roomID, before, cacheSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		if err := rows.Scan(&msg.UserID, &msg.Username, &msg.RoomID, &msg.Message, &msg.Timestamp); err != nil {
			return nil, err
		}
		messages = append(messages, &msg)
	}

	return messages, rows.Err()
}

func (r *messageRepo) warmCache(ctx context.Context, key string, messages []*domain.Message) {
	pipe := r.rdb.Pipeline()
	for i := range messages {
		data, err := json.Marshal(messages[i])
		if err != nil {
			continue
		}
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(messages[i].Timestamp.Unix()), Member: string(data)})
	}
	pipe.Expire(ctx, key, cacheTTL)
	pipe.Exec(ctx)
}

func deserializeMessage(raw []string) ([]*domain.Message, error) {
	messages := make([]*domain.Message, len(raw))
	for i := range raw {
		msg := &domain.Message{}
		if err := json.Unmarshal([]byte(raw[i]), &msg); err != nil {
			return nil, err
		}
		messages[i] = msg
	}
	return messages, nil
}
