package components

import (
	"context"
	"log"
	"log/slog"
	"presence_service/internal/repository"
	"presence_service/internal/service"
	"presence_service/pkg/config"
	"presence_service/pkg/service_logger"
	"time"

	"github.com/redis/go-redis/v9"
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

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Redis.Addrs,
		Password: cfg.Redis.Password,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}

	repo := repository.NewPresenceRepo(rdb, cfg.OnlineTTL)
	svc := service.NewPresenceService(repo, logger)

	return &Components{
		Svc:    svc,
		Logger: logger,
		Rdb:    rdb,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Rdb.Close()
}
