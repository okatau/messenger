package repository

import (
	"context"
	"testing"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getMessageFromRedis(t *testing.T, rdb *redis.Client, roomID string) []*domain.Message {
	t.Helper()
	key := cacheKey(roomID)
	cached, err := rdb.ZRangeArgs(t.Context(), redis.ZRangeArgs{Key: key, Start: 0, Stop: cacheSize - 1, Rev: true}).Result()
	require.NoError(t, err)
	if len(cached) > 0 {
		msgs, err := deserializeMessage(cached)
		require.NoError(t, err)
		return msgs
	}
	t.Error("did't get messages from redis")
	return nil
}

func getMessageFromPG(t *testing.T, pool *pgxpool.Pool, roomID string) []*domain.Message {
	t.Helper()
	query := `
		SELECT m.sender_id, u.name, m.room_id, m.body, m.created_at
		FROM messages m
		JOIN users u ON u.id = m.sender_id
		WHERE m.room_id = $1
		ORDER BY m.created_at DESC
		LIMIT $2
	`
	rows, err := pool.Query(t.Context(), query, roomID, cacheSize)
	require.NoError(t, err)
	defer rows.Close()

	var messages []*domain.Message
	for rows.Next() {
		var msg domain.Message
		err := rows.Scan(&msg.UserID, &msg.Username, &msg.RoomID, &msg.Message, &msg.Timestamp)
		require.NoError(t, err)
		messages = append(messages, &msg)
	}

	return messages
}
func Test_MessageRepo(t *testing.T) {
	ss := setup(t)
	redis, cleanupRedis := startRedis(t)
	defer func() {
		cleanupRedis()
	}()

	repo := NewMessageRepository(ss.pool, redis)

	msg := &domain.Message{
		RoomID:  ss.roomID,
		UserID:  ss.userID,
		Message: "hello",
	}
	ctx := context.Background()

	err := repo.WriteMessage(ctx, msg)
	require.NoError(t, err)

	msgPG := getMessageFromPG(t, ss.pool, ss.roomID)
	msgR := getMessageFromRedis(t, redis, ss.roomID)
	assert.Equal(t, len(msgPG), len(msgR))
	assert.Equal(t, msgPG[0].UserID, msgR[0].UserID)
}
