package repository

import (
	"context"
	"friends_service/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
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

func setup(t *testing.T) (repo FriendshipRepository, aliceID, bobID string) {
	t.Helper()

	pool, cleanup := startPostgres(t)
	t.Cleanup(func() {
		cleanup()
	})

	repo = NewFriendshipRepository(pool)

	aliceID = createUser(t, t.Context(), pool, "alice")
	bobID = createUser(t, t.Context(), pool, "bob")
	return
}

func Test_AddFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
}

func Test_AddFriend_Duplicate(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	err = repo.AddFriend(t.Context(), aliceID, bobID)
	assert.ErrorIs(t, err, domain.ErrFriendReqAlreadyExists)
}

func Test_AcceptFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	accepted, err := repo.AcceptFriend(t.Context(), bobID, aliceID)
	require.NoError(t, err)
	assert.Equal(t, accepted, true)
}

func Test_AcceptFriend_NoInvite(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	accepted, err := repo.AcceptFriend(t.Context(), bobID, aliceID)
	require.NoError(t, err)
	assert.Equal(t, accepted, false)
}

func Test_DeclineFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	accepted, err := repo.DeclineFriend(t.Context(), bobID, aliceID)
	require.NoError(t, err)
	assert.Equal(t, accepted, true)
}

func Test_DeclineFriend_NoInvite(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	accepted, err := repo.DeclineFriend(t.Context(), bobID, aliceID)
	require.NoError(t, err)
	assert.Equal(t, accepted, false)
}

func Test_CancelFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	accepted, err := repo.CancelFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
	assert.Equal(t, accepted, true)
}

func Test_CancelFriend_NoInvite(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	accepted, err := repo.CancelFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
	assert.Equal(t, accepted, false)
}

func Test_RemoveFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	err := repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	_, err = repo.AcceptFriend(t.Context(), bobID, aliceID)
	require.NoError(t, err)

	accepted, err := repo.RemoveFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
	assert.Equal(t, accepted, true)
}

func Test_RemoveFriend_NoFriend(t *testing.T) {
	repo, aliceID, bobID := setup(t)

	accepted, err := repo.RemoveFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
	assert.Equal(t, accepted, false)

	err = repo.AddFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)

	accepted, err = repo.RemoveFriend(t.Context(), aliceID, bobID)
	require.NoError(t, err)
	assert.Equal(t, accepted, false)
}
