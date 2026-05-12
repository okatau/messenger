package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"friends_service/internal/repository"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
	"friends_service/pkg/token_manager"
	"log"
	"log/slog"
	"strings"

	"github.com/go-redis/redis_rate/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Env          string                `yaml:"env" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	Auth         config.AuthConfig
	ServerConfig config.HTTPConfig `yaml:"http"`
	Limits       RateLimits        `yaml:"limits"`
	Redis        config.RedisConfig
}

type RateLimits struct {
	SearchLimit int `yaml:"search" env-default:"20"`
	AddLimit    int `yaml:"add" env-default:"10"`
}

type Components struct {
	Svc          service.Friendship
	Logger       *slog.Logger
	TokenManager *token_manager.TokenManager
	Postgres     *pgxpool.Pool
	Limiter      *redis_rate.Limiter
}

func InitComponents(ctx context.Context, cfg *Config) *Components {
	dsn := getPostgresDSN(cfg.Postgres)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    cfg.Redis.Addrs,
		Password: cfg.Redis.Password,
	})
	if err = rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping failed: %v", err)
	}
	limiter := redis_rate.NewLimiter(rdb)

	logger := service_logger.InitLogger(cfg.Env)
	pemBytes, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(cfg.Auth.PublicKeyPEMBase64, "\n", ""))
	if err != nil {
		log.Fatalf("error decoding public pem %v", err)
	}

	manager, err := token_manager.NewTokenManager(pemBytes, []byte{}, cfg.Auth.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	userRepo := repository.NewUserRepository(pool)
	friendshipRepo := repository.NewFriendshipRepository(pool)
	svc := service.NewFriendshipService(userRepo, friendshipRepo, logger)

	return &Components{
		Svc:          svc,
		Logger:       logger,
		TokenManager: manager,
		Postgres:     pool,
		Limiter:      limiter,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
