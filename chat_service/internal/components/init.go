package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"time"

	"chat_service/internal/repository"
	"chat_service/internal/service"
	"chat_service/pkg/config"
	"chat_service/pkg/logger"
	"chat_service/pkg/token_manager"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type Config struct {
	Env          string                `yaml:"env" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	Redis        config.RedisConfig
	Auth         AuthConfig
	ServerConfig config.HTTPConfig `yaml:"http"`
}

type AuthConfig struct {
	AccessTokenTTL time.Duration `yaml:"access_token_ttl" env-default:"15m"`
	// RefreshTokenTTL    time.Duration `yaml:"refresh_token_ttl" env-default:"720h"` // 30 days
	PublicKeyPEMBase64 string `env:"AUTH_PUBLIC_PEM_BASE64" env-required:"true"`
}

type Components struct {
	Postgres     *pgxpool.Pool
	Redis        redis.UniversalClient
	Hub          service.Hub
	TokenManager *token_manager.TokenManager
	Logger       *slog.Logger
}

func InitComponents(ctx context.Context, hubCtx context.Context, cfg *Config) *Components {
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

	logger := logger.InitLogger(cfg.Env)

	pemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}

	manager, err := token_manager.NewTokenManager(pemBytes, []byte{}, cfg.Auth.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	roomRepo := repository.NewRoomRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	msgRepo := repository.NewMessageRepository(pool, rdb)

	hub := service.NewHub(hubCtx, userRepo, roomRepo, msgRepo, logger)

	return &Components{
		Postgres:     pool,
		Redis:        rdb,
		Hub:          hub,
		TokenManager: manager,
		Logger:       logger,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
	c.Redis.Close()
	c.Hub.Shutdown(ctx)
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
