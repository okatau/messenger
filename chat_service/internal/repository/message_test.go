package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"testing"
	"time"

	"chat_service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func Test_GetMessageBefore_Redis(t *testing.T) {
	ss := setup(t)

	rdb, cleanupRedis := startRedis(t)
	defer func() {
		cleanupRedis()
	}()

	repo := NewMessageRepository(ss.pool, rdb)

	roomID := uuid.NewString()
	before := time.Now()
	addNMessagesRedis(t, roomID, 50, rdb, before)

	msgs, err := repo.GetMessagesBefore(t.Context(), roomID, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 50, len(msgs))
}

func Test_GetMessageBefore_PG(t *testing.T) {
	ss := setup(t)

	rdb, cleanupRedis := startRedis(t)
	defer func() {
		cleanupRedis()
	}()

	repo := NewMessageRepository(ss.pool, rdb)

	before := time.Now()
	addNMessagesPG(t, ss.roomID, ss.userID, 50, ss.pool, before)

	msgs, err := repo.GetMessagesBefore(t.Context(), ss.roomID, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 50, len(msgs))
}

func Test_GetMessageBefore_PartialCache(t *testing.T) {
	ss := setup(t)

	rdb, cleanupRedis := startRedis(t)
	defer cleanupRedis()

	repo := NewMessageRepository(ss.pool, rdb)

	roomID, userID := createRoom(t, t.Context(), ss.pool, "room1")

	before := time.Now()
	addNMessagesRedis(t, roomID, 15, rdb, before)
	addNMessagesPG(t, roomID, userID, 35, ss.pool, before.Add(-15*time.Minute))

	msgs, err := repo.GetMessagesBefore(t.Context(), roomID, time.Now().Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 50, len(msgs))
}

func Test_GetMessageBefore_PartialCache_NoDuplicates(t *testing.T) {
	ss := setup(t)

	rdb, cleanupRedis := startRedis(t)
	defer cleanupRedis()

	repo := NewMessageRepository(ss.pool, rdb)

	roomID, userID := createRoom(t, t.Context(), ss.pool, "room_overlap")

	// 10 сообщений в Redis: base, base-1min, ..., base-9min
	// 50 сообщений в PG: те же временные метки + 40 более старых (overlap на первых 10)
	base := time.Now().Truncate(time.Minute)
	addNMessagesRedis(t, roomID, 10, rdb, base)
	addNMessagesPG(t, roomID, userID, 50, ss.pool, base)

	msgs, err := repo.GetMessagesBefore(t.Context(), roomID, base.Add(time.Minute))
	require.NoError(t, err)
	assert.Equal(t, 50, len(msgs))

	seen := make(map[int64]bool)
	for _, m := range msgs {
		ts := m.Timestamp.Unix()
		assert.False(t, seen[ts], "duplicate message at unix=%d", ts)
		seen[ts] = true
	}
}

func addNMessagesRedis(t *testing.T, roomID string, n int, rdb *redis.Client, before time.Time) {
	t.Helper()

	pipe := rdb.Pipeline()
	key := cacheKey(roomID)
	for i := range n {
		raw := make([]byte, 5)
		rand.Read(raw)
		name := hex.EncodeToString(raw)
		msg := domain.Message{
			Username:  name,
			Message:   fmt.Sprintf("%s send ith message - %d", name, i),
			RoomID:    roomID,
			Timestamp: before.Add(-time.Duration(i) * time.Minute),
		}

		data, _ := json.Marshal(msg)

		pipe.ZAdd(t.Context(), key, redis.Z{Score: float64(msg.Timestamp.Unix()), Member: string(data)})
		pipe.Expire(t.Context(), key, cacheTTL)
	}

	_, err := pipe.Exec(t.Context())
	require.NoError(t, err)
}

func addNMessagesPG(t *testing.T, roomID, userID string, n int, pool *pgxpool.Pool, before time.Time) {
	t.Helper()

	tx, err := pool.Begin(t.Context())
	if err != nil {
		log.Fatal(err)
	}
	defer tx.Rollback(t.Context())

	batch := &pgx.Batch{}

	queue := `
		INSERT INTO messages (room_id, sender_id, body, created_at)
		VALUES ($1, $2, $3, $4)
	`

	for i := range n {
		raw := make([]byte, 5)
		rand.Read(raw)
		name := hex.EncodeToString(raw)

		batch.Queue(
			queue,
			roomID,
			userID,
			fmt.Sprintf("%s send ith message - %d", name, i),
			before.Add(-time.Duration(i)*time.Minute),
		)
	}

	br := tx.SendBatch(t.Context(), batch)

	for i := 0; i < batch.Len(); i++ {
		if _, err := br.Exec(); err != nil {
			br.Close()
			t.Fatal(err)
		}
	}
	br.Close()

	require.NoError(t, tx.Commit(t.Context()))
}
