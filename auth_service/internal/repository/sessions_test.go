package repository

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"golang.org/x/crypto/bcrypt"
)

var (
	randomID string = uuid.NewString()

	aliceID    string
	aliceName  = "alice"
	aliceMail  = "alice@mail.com"
	alicePW    = "alice"
	alicePH, _ = bcrypt.GenerateFromPassword([]byte(alicePW), bcrypt.DefaultCost)
)

const (
	sessionTTL = 30 * 24 * time.Hour
)

func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()

	ctr, err := tcpostgres.Run(t.Context(),
		"postgres:16-alpine",
		tcpostgres.WithDatabase("test_auth"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)

	dsn, err := ctr.ConnectionString(t.Context(), "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(t.Context(), dsn)
	if err != nil {
		ctr.Terminate(t.Context())
		t.Fatalf("connect to db: %v", err)
	}

	runMigrations(t, pool)
	aliceID, err = createUser(t.Context(), t, pool, aliceName, aliceMail, string(alicePH))
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
		ctr.Terminate(t.Context())
	})

	return pool
}

func runMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	migrationsDir := "../../../chat_service/internal/db/migrations"

	_, err := pool.Exec(t.Context(), `CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)
	require.NoError(t, err)

	entries, err := os.ReadDir(migrationsDir)
	require.NoError(t, err, "read migrations dir")

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}

		sql, err := os.ReadFile(filepath.Join(migrationsDir, entry.Name()))
		require.NoError(t, err, "read migration: %s", entry.Name())

		_, err = pool.Exec(t.Context(), string(sql))
		require.NoError(t, err, "apply migration: %s", entry.Name())
	}
}

func createSession(ctx context.Context, t *testing.T, repo SessionRepository, userID, name, refreshToken string) {
	t.Helper()
	err := repo.CreateSession(ctx, userID, name, refreshToken, time.Now().Add(sessionTTL))
	require.NoError(t, err)
}

func createUser(ctx context.Context, t *testing.T, pool *pgxpool.Pool, name, email, passwordHash string) (string, error) {
	t.Helper()
	uRepo := NewUserRepository(pool)
	user, err := uRepo.CreateUser(ctx, name, email, passwordHash)
	return user.ID, err
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func Test_CreateSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)

		refreshToken, _ := generateRefreshToken()
		err := sRepo.CreateSession(t.Context(), aliceID, "CreateSession_test", refreshToken, time.Now().Add(sessionTTL))
		require.NoError(t, err)

		session, err := sRepo.GetSessionByToken(t.Context(), refreshToken)
		require.NoError(t, err)
		require.Equal(t, aliceID, session.UserID)
	})

	t.Run("add duplicate", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)

		refreshToken, _ := generateRefreshToken()
		err := sRepo.CreateSession(t.Context(), aliceID, "CreateSession_test", refreshToken, time.Now().Add(sessionTTL))
		require.NoError(t, err)
		err = sRepo.CreateSession(t.Context(), aliceID, "CreateSession_test", refreshToken, time.Now().Add(sessionTTL))

		var pgErr *pgconn.PgError
		require.ErrorAs(t, err, &pgErr)
		assert.Equal(t, pgErr.Code, "23505") // unique violation
	})
}

func Test_DeleteSession(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)

		refreshToken, _ := generateRefreshToken()

		createSession(t.Context(), t, sRepo, aliceID, aliceName, refreshToken)

		_, err := sRepo.DeleteSession(t.Context(), refreshToken)
		require.NoError(t, err)

		session, err := sRepo.GetSessionByToken(t.Context(), refreshToken)
		require.NoError(t, err)
		require.Nil(t, session)
	})

	t.Run("session does not exist", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)
		refreshToken, _ := generateRefreshToken()

		session, err := sRepo.DeleteSession(t.Context(), refreshToken)
		require.NoError(t, err)
		require.Nil(t, session)
	})
}

func Test_DeleteSessionsByUserID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)
		refreshToken, _ := generateRefreshToken()

		createSession(t.Context(), t, sRepo, aliceID, aliceName, refreshToken)

		session, err := sRepo.DeleteSessionsByUserID(t.Context(), aliceID)
		require.NoError(t, err)
		require.NotNil(t, session)
	})

	t.Run("session does not exist", func(t *testing.T) {
		pool := startPostgres(t)

		sRepo := NewSessionRepository(pool)
		if aliceID == randomID {
			t.Errorf("identical ids aliceID:%v randomID:%v", aliceID, randomID)
		}

		session, err := sRepo.DeleteSessionsByUserID(t.Context(), randomID)
		require.NoError(t, err)
		require.Nil(t, session)
	})
}
