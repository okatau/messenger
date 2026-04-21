package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

var (
	aliceID  string
	randomID string = uuid.NewString()
)

const (
	aliceName         = "alice"
	aliceEmail        = "alice@mail.com"
	alicePasswordHash = "aaa"

	sessionTTL = 30 * 24 * time.Hour
)

// Add alice as user by default
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
	aliceID, err = createUser(t, ctx, pool, aliceName, aliceEmail, alicePasswordHash)
	require.NoError(t, err)

	return pool, func() {
		pool.Close()
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

func createSession(t *testing.T, ctx context.Context, repo SessionRepository, userID, name, refreshToken string) {
	t.Helper()
	err := repo.CreateSession(ctx, userID, name, refreshToken, time.Now().Add(sessionTTL))
	require.NoError(t, err)
}

func createUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool, name, email, passwordHash string) (string, error) {
	t.Helper()
	uRepo := NewUserRepository(pool)
	user, err := uRepo.CreateUser(ctx, name, email, passwordHash)
	return user.ID, err
}

func Test_CreateSession(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	sRepo := NewSessionRepository(pool)

	ctx := context.Background()

	refreshToken, _ := generateRefreshToken()
	err := sRepo.CreateSession(ctx, aliceID, "room1", refreshToken, time.Now().Add(sessionTTL))
	require.NoError(t, err)

	session, err := sRepo.GetSessionByToken(ctx, refreshToken)
	require.NoError(t, err)
	require.Equal(t, aliceID, session.UserID)
}

func Test_DeleteSession(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	sRepo := NewSessionRepository(pool)

	refreshToken, _ := generateRefreshToken()
	ctx := context.Background()

	createSession(t, ctx, sRepo, aliceID, aliceName, refreshToken)

	session, err := sRepo.DeleteSession(ctx, refreshToken)
	require.NoError(t, err)
	require.NotNil(t, session, "session is nil")
}

func Test_DeleteSession_NoRows(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	sRepo := NewSessionRepository(pool)
	refreshToken, _ := generateRefreshToken()
	ctx := context.Background()

	session, err := sRepo.DeleteSession(ctx, refreshToken)
	require.NoError(t, err)
	require.Nil(t, session, "session is not nil")
}

func Test_DeleteSessionsByUserID(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	sRepo := NewSessionRepository(pool)
	refreshToken, _ := generateRefreshToken()
	ctx := context.Background()

	createSession(t, ctx, sRepo, aliceID, aliceName, refreshToken)

	session, err := sRepo.DeleteSessionsByUserID(ctx, aliceID)
	require.NoError(t, err)
	require.NotNil(t, session, "session is nil")
}

func Test_DeleteSessionsByUserID_NoRows(t *testing.T) {
	pool, cleanup := startPostgres(t)
	defer cleanup()

	sRepo := NewSessionRepository(pool)
	ctx := context.Background()
	if aliceID == randomID {
		t.Errorf("identical ids aliceID:%v randomID:%v", aliceID, randomID)
	}

	session, err := sRepo.DeleteSessionsByUserID(ctx, randomID)
	require.NoError(t, err)
	require.Nil(t, session, "session is nil")
}
