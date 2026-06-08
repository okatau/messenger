package components

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"

	"presence_service/internal/repository"
	"presence_service/internal/service"
	"presence_service/pkg/config"
	"presence_service/pkg/service_logger"
)

type Config struct {
	Env          string `yaml:"env" env-default:"local"`
	Redis        config.RedisConfig
	ServerConfig config.ServerConfig
	OnlineTTL    time.Duration `yaml:"online_ttl" env-default:"60s"`
}

type Components struct {
	Svc    service.Presence
	Logger *slog.Logger
	Rdb    redis.UniversalClient
}

func InitComponents(ctx context.Context, cfg *Config) *Components {
	logger := service_logger.InitLogger(cfg.Env)

	rdb := initRedis(ctx, cfg.Redis)

	repo := repository.NewPresenceRepo(rdb, cfg.OnlineTTL)
	svc := service.New(repo, logger)

	return &Components{
		Svc:    svc,
		Logger: logger,
		Rdb:    rdb,
	}
}

func (c *Components) Shutdown() {
	//nolint:errcheck // no need to check err
	c.Rdb.Close()
}

func initRedis(ctx context.Context, cfg config.RedisConfig) redis.UniversalClient {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Addrs,
		Password: cfg.Password,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	return rdb
}
