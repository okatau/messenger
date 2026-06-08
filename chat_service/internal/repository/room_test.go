package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chat_service/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

var (
	roomName = "room-1"

	aliceName = "alice"
	bobName   = "bob"
)

func startPostgres(t *testing.T) *pgxpool.Pool {
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

	t.Cleanup(func() {
		pool.Close()
		ctr.Terminate(ctx)
	})

	return pool
}

func startRedis(t *testing.T) *redis.Client {
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

	t.Cleanup(func() {
		redisClient.Close()
		ctr.Terminate(ctx)
	})

	return redisClient
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	ctx := context.Background()

	migrationsDir := "../db/migrations"

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

func createUser(t *testing.T, pool *pgxpool.Pool, name string) string {
	t.Helper()
	query := `
		INSERT INTO users (name, email, password_hash)
		VALUES ($1, $1, $1)
		RETURNING (id)
	`

	var id string
	err := pool.QueryRow(t.Context(), query, name).Scan(&id)
	require.NoError(t, err)
	return id
}

func createRoom(t *testing.T, pool *pgxpool.Pool, name, userID string) (roomID string) {
	t.Helper()

	var room domain.Room
	err := pool.QueryRow(t.Context(), `
		INSERT INTO rooms(name, created_by)
		VALUES ($1, $2)
		RETURNING id, name, created_by, created_at
	`, "room_"+name, userID).Scan(&room.ID, &room.Name, &room.CreatedBy, &room.CreatedAt)
	require.NoError(t, err)

	roomID = room.ID
	return
}

func Test_Room_CreateRoom(t *testing.T) {
	pool := startPostgres(t)
	repo := NewRoomRepository(pool)
	userID := createUser(t, pool, aliceName)

	room, err := repo.CreateRoom(t.Context(), roomName, userID)
	require.NoError(t, err)
	assert.Equal(t, *room.Name, roomName)

	rooms, err := repo.GetRoomsByUserID(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, rooms[0].ID, room.ID)
}

func Test_Room_DeleteRoom(t *testing.T) {
	t.Run("Deleting group room", func(t *testing.T) {
		pool := startPostgres(t)
		repo := NewRoomRepository(pool)

		userID := createUser(t, pool, aliceName)
		roomID := createRoom(t, pool, roomName, userID)
		room, err := repo.DeleteRoom(t.Context(), roomID)

		require.NoError(t, err)
		assert.Equal(t, room.ID, roomID)
	})

	t.Run("Deleting direct room", func(t *testing.T) {
		pool := startPostgres(t)
		repo := NewRoomRepository(pool)

		alice := createUser(t, pool, aliceName)
		bob := createUser(t, pool, bobName)
		room, err := repo.CreateDM(t.Context(), alice, bob)
		require.NoError(t, err)

		room, err = repo.DeleteRoom(t.Context(), room.ID)
		require.NoError(t, err)

		rooms, err := repo.GetDMsByUserID(t.Context(), alice)
		require.NoError(t, err)
		assert.Len(t, rooms, 0)
	})
}

func Test_Room_AddUser(t *testing.T) {
	pool := startPostgres(t)
	repo := NewRoomRepository(pool)

	alice := createUser(t, pool, aliceName)
	bob := createUser(t, pool, bobName)
	room := createRoom(t, pool, roomName, alice)

	err := repo.AddUser(t.Context(), bob, room)
	require.NoError(t, err)

	exists, err := repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	assert.Equal(t, exists, true)
}

func Test_Room_RemoveUser(t *testing.T) {
	pool := startPostgres(t)
	repo := NewRoomRepository(pool)

	alice := createUser(t, pool, aliceName)
	bob := createUser(t, pool, bobName)
	room := createRoom(t, pool, roomName, alice)
	err := repo.AddUser(t.Context(), bob, room)
	require.NoError(t, err)

	exists, err := repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	assert.Equal(t, exists, true)

	err = repo.RemoveUser(t.Context(), bob, room)
	require.NoError(t, err)

	member, err := repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	assert.False(t, member)
}

func Test_Room_CreateDM(t *testing.T) {
	pool := startPostgres(t)
	repo := NewRoomRepository(pool)

	aliceID := createUser(t, pool, aliceName)
	bobID := createUser(t, pool, bobName)
	_, err := repo.CreateDM(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	rooms, err := repo.GetDMsByUserID(t.Context(), aliceID)
	require.NoError(t, err)
	assert.Len(t, rooms, 1)
	assert.Equal(t, rooms[0].CreatedBy, aliceID)
}
