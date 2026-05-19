package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
)

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

func Test_Add(t *testing.T) {
	rdb, cleanup := startRedis(t)
	defer cleanup()
	t.Run("Successfully added status", func(t *testing.T) {
		repo := NewPresenceRepo(rdb, 10*time.Second)

		key := "test:case-1"
		k, v := "msgKey", "added successfully"

		err := repo.Add(t.Context(), key, map[string]string{k: v})
		require.NoError(t, err)

		res, err := repo.Get(t.Context(), key)
		require.NoError(t, err)

		fmt.Println(res, err)

		msg, ok := res[k]
		assert.True(t, ok)
		assert.Equal(t, v, msg)
	})

	t.Run("Successfully deleted after expire", func(t *testing.T) {
		repo := NewPresenceRepo(rdb, 2*time.Second)

		key := "test:case-2"
		k, v := "msgKey", "added successfully"

		err := repo.Add(t.Context(), key, map[string]string{k: v})
		require.NoError(t, err)

		res, err := repo.Get(t.Context(), key)
		require.NoError(t, err)

		msg, ok := res[k]
		assert.True(t, ok)
		assert.Equal(t, v, msg)

		time.Sleep(2 * time.Second)

		res, err = repo.Get(t.Context(), key)
		require.NoError(t, err)

		_, ok = res[k]
		assert.False(t, ok)
	})

	t.Run("Adding same key twice", func(t *testing.T) {
		repo := NewPresenceRepo(rdb, 10*time.Second)

		key := "test:case-3"
		k, v := "msgKey", "added successfully"

		err := repo.Add(t.Context(), key, map[string]string{k: v})
		require.NoError(t, err)

		err = repo.Add(t.Context(), key, map[string]string{k: v})
		require.NoError(t, err)
	})
}

func Test_Update(t *testing.T) {
	rdb, cleanup := startRedis(t)
	defer cleanup()

	t.Run("Successfully updated status", func(t *testing.T) {
		repo := NewPresenceRepo(rdb, 20*time.Second)

		key := "test:case-1"
		k, v := "status", "online"

		err := repo.Add(t.Context(), key, map[string]string{k: v})
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		firstTS, err := rdb.TTL(t.Context(), key).Result()
		require.NoError(t, err)
		assert.Greater(t, firstTS.Milliseconds(), int64(0))

		res, err := repo.Update(t.Context(), key)
		require.NoError(t, err)
		assert.True(t, res)

		secondTS, err := rdb.TTL(t.Context(), key).Result()
		require.NoError(t, err)
		assert.Greater(t, secondTS, firstTS)
	})

	t.Run("Updating non exist status", func(t *testing.T) {
		repo := NewPresenceRepo(rdb, 20*time.Second)

		key := "test:case-2"

		res, err := repo.Update(t.Context(), key)
		require.NoError(t, err)
		assert.False(t, res)
	})
}
