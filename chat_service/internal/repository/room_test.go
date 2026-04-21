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

func Test_Room_CreateRoom(t *testing.T) {
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

func Test_Room_DeleteRoom(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	room, err := ss.repo.DeleteRoom(ctx, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, room.ID, ss.roomID)
}

func Test_Room_AddUser(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	bobID := createUser(t, ctx, ss.pool, "bob")
	err := ss.repo.AddUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, exists, true)
}

func RemoveUser(t *testing.T) {
	ss := setup(t)
	ctx := context.Background()

	bobID := createUser(t, ctx, ss.pool, "bob")
	err := ss.repo.AddUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	exists, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	assert.Equal(t, exists, true)

	err = ss.repo.RemoveUser(ctx, bobID, ss.roomID)
	require.NoError(t, err)

	member, err := ss.repo.IsMember(ctx, bobID, ss.roomID)
	require.NoError(t, err)
	if member {
		t.Error("user had not been deleted")
	}
}
