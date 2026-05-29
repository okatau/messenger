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
	require.NoError(t, err, "error adding user")
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
	require.NoError(t, err, "error adding user")

	roomID = room.ID
	return
}

type Setup struct {
	repo RoomRepository
	pool *pgxpool.Pool
}

func setup(t *testing.T) *Setup {
	pool, cleanupPg := startPostgres(t)

	repo := NewRoomRepository(pool)

	t.Cleanup(func() {
		cleanupPg()
	})

	return &Setup{
		repo: repo,
		pool: pool,
	}
}

func Test_Room_CreateRoom(t *testing.T) {
	ss := setup(t)
	userID := createUser(t, ss.pool, "user")
	name := "room1"

	room, err := ss.repo.CreateRoom(t.Context(), name, userID)
	require.NoError(t, err)
	assert.Equal(t, *room.Name, name)

	rooms, err := ss.repo.GetRoomsByUserID(t.Context(), userID)
	require.NoError(t, err)
	assert.Equal(t, rooms[0].ID, room.ID)
}

func Test_Room_DeleteRoom(t *testing.T) {
	ss := setup(t)

	t.Run("Deleting group room", func(t *testing.T) {
		userID := createUser(t, ss.pool, "user_1")
		roomID := createRoom(t, ss.pool, "delete_room_1", userID)
		room, err := ss.repo.DeleteRoom(t.Context(), roomID)

		require.NoError(t, err)
		assert.Equal(t, room.ID, roomID)
	})

	t.Run("Deleting direct room", func(t *testing.T) {
		alice := createUser(t, ss.pool, "alice")
		bob := createUser(t, ss.pool, "bob")
		room, err := ss.repo.CreateDM(t.Context(), alice, bob)
		require.NoError(t, err)

		room, err = ss.repo.DeleteRoom(t.Context(), room.ID)
		require.NoError(t, err)

		rooms, err := ss.repo.GetDMsByUserID(t.Context(), alice)
		require.NoError(t, err)
		assert.Len(t, rooms, 0)
	})
}

func Test_Room_AddUser(t *testing.T) {
	ss := setup(t)

	alice := createUser(t, ss.pool, "alice")
	bob := createUser(t, ss.pool, "bob")
	room := createRoom(t, ss.pool, "add_user", alice)

	err := ss.repo.AddUser(t.Context(), bob, room)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	assert.Equal(t, exists, true)
}

func Test_Room_RemoveUser(t *testing.T) {
	ss := setup(t)

	alice := createUser(t, ss.pool, "alice")
	bob := createUser(t, ss.pool, "bob")
	room := createRoom(t, ss.pool, "add_user", alice)
	err := ss.repo.AddUser(t.Context(), bob, room)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	assert.Equal(t, exists, true)

	err = ss.repo.RemoveUser(t.Context(), bob, room)
	require.NoError(t, err)

	member, err := ss.repo.IsMember(t.Context(), bob, room)
	require.NoError(t, err)
	if member {
		t.Error("user had not been deleted")
	}
}

func Test_Room_CreateDM(t *testing.T) {
	ss := setup(t)

	aliceID := createUser(t, ss.pool, "alice")
	bobID := createUser(t, ss.pool, "bob")
	_, err := ss.repo.CreateDM(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	rooms, err := ss.repo.GetDMsByUserID(t.Context(), aliceID)
	require.NoError(t, err)
	assert.Len(t, rooms, 1)
	assert.Equal(t, rooms[0].CreatedBy, aliceID)
}
