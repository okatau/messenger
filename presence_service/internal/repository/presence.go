package repository

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type PresenceRepository interface {
	Add(ctx context.Context, key string, metadata map[string]string) error
	Update(ctx context.Context, key string) (bool, error)
	Get(ctx context.Context, key string) (map[string]string, error)
	GetBulk(ctx context.Context, keys []string) ([]*redis.MapStringStringCmd, error)
}

type presenceRepo struct {
	rdb       redis.UniversalClient
	onlineTTL time.Duration
}

func NewPresenceRepo(rdb redis.UniversalClient, onlineTTL time.Duration) PresenceRepository {
	return &presenceRepo{
		rdb:       rdb,
		onlineTTL: onlineTTL,
	}
}

func (r *presenceRepo) Add(ctx context.Context, key string, metadata map[string]string) error {
	pipe := r.rdb.Pipeline()
	pipe.HMSet(ctx, key, metadata)
	pipe.Expire(ctx, key, r.onlineTTL)
	_, err := pipe.Exec(ctx)

	return err
}

func (r *presenceRepo) Update(ctx context.Context, key string) (bool, error) {
	return r.rdb.Expire(ctx, key, r.onlineTTL).Result()
}

func (r *presenceRepo) Get(ctx context.Context, key string) (map[string]string, error) {
	return r.rdb.HGetAll(ctx, key).Result()
}

func (r *presenceRepo) GetBulk(ctx context.Context, keys []string) ([]*redis.MapStringStringCmd, error) {
	pipe := r.rdb.Pipeline()

	cmds := make([]*redis.MapStringStringCmd, len(keys))
	for i := range cmds {
		cmds[i] = pipe.HGetAll(ctx, keys[i])
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, err
	}
	return cmds, nil
}
