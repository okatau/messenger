package repository

import (
	"chat_service/internal/domain"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4/testutils/assert"
	"github.com/go-jose/go-jose/v4/testutils/require"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

func startPostgres(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test_auth"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		ctr.Terminate(ctx)
		t.Fatalf("connect to db: %v", err)
	}

	runMigrations(t, pool)
	require.NoError(t, err)

	return pool, func() {
		pool.Close()
		ctr.Terminate(ctx)
	}
}

func startRedis(t *testing.T) (*redis.Client, func()) {
	t.Helper()

	ctx := context.Background()
	ctr, err := tcredis.Run(ctx,
		"redis:7-alpine",
		tcredis.WithSnapshotting(10, 1),
		tcredis.WithLogLevel(tcredis.LogLevelVerbose),
	)
	require.NoError(t, err)

	dsn, err := ctr.ConnectionString(ctx)
	require.NoError(t, err)

	opt, err := redis.ParseURL(dsn)
	require.NoError(t, err)

	redisClient := redis.NewClient(opt)

	return redisClient, func() {
		redisClient.Close()
		ctr.Terminate(ctx)
	}
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	migrationsDir := "../../../migrations"

	_, err := pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)
	require.NoError(t, err)

	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err, "read migrations dir")

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		require.NoError(t, err, "read migration: %s", entry.Name())

		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply migration: %s", entry.Name())
	}
}

func createUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) string {
	t.Helper()
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $1, $1)
		RETURNING (id)
	`

	var id string
	err := pool.QueryRow(ctx, query, name).Scan(&id)
	require.NoError(t, err, "error adding user")
	return id
}

func createRoom(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name string) (roomID, userID string) {
	t.Helper()
	userID = createUser(t, ctx, pool, "user_"+name)

	var room domain.Room
	err := pool.QueryRow(ctx, `
		INSERT INTO rooms(name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at
	`, "room_"+name, userID).Scan(&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt)
	require.NoError(t, err, "error adding user")

	roomID = room.ID
	return
}

type Setup struct {
	userID string
	roomID string
	repo   RoomRepository
	pool   *pgxpool.Pool
	redis  *redis.Client
}

func setup(t *testing.T) *Setup {
	pool, cleanupPg := startPostgres(t)

	repo := NewRoomRepository(pool)
	name := "initial"
	roomID, userID := createRoom(t, t.Context(), pool, name)

	t.Cleanup(func() {
		cleanupPg()
	})

	return &Setup{
		userID: userID,
		roomID: roomID,
		repo:   repo,
		pool:   pool,
	}
}

func Test_RoomRepo_CreateRoom(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()
	name := "room1"

	room, err := ss.repo.CreateRoom(ctx, name, ss.userID)
	require.NoError(t, err)
	assert.Equal(t, room.Name, name)

	rooms, err := ss.repo.GetRoomsByUserID(ctx, ss.userID)
	require.NoError(t, err)
	assert.Equal(t, rooms[0].ID, room.ID)
}

func Test_RoomRepo_DeleteRoom(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	room, err := ss.repo.DeleteRoom(ctx, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, ss.roomID)
}

func Test_RoomRepo_AddUser(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	bobID := createUser(t, ctx, ss.pool, "bob")
	err := ss.repo.AddUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, exists, true)
}

func Test_RoomRepo_DeleteUser(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	bobID := createUser(t, ctx, ss.pool, "bob")
	err := ss.repo.AddUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, exists, true)

	err = ss.repo.DeleteUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	member, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	if member {
		t.Error("user had not been deleted")
	}
}

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
