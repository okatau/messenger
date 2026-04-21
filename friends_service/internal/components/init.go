package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"friends_service/internal/repository"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/logger"
	"friends_service/pkg/token_manager"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Env      string                `env:"ENV" env-default:"local"`
	Host     string                `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port     int                   `env:"SERVER_FRIENDS_PORT" env-default:"8083"`
	Postgres config.PostgresConfig `env-prefix:"PG_"`
	Auth     AuthConfig            `env-prefix:"AUTH_"`
}

type AuthConfig struct {
	PublicKeyPEMBase64 string `env:"PUBLIC_PEM_BASE64" env-default:""`
}

type Components struct {
	Svc          service.Friendship
	Logger       *slog.Logger
	TokenManager *token_manager.TokenManager
	Postgres     *pgxpool.Pool
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

	logger := logger.InitLogger(cfg.Env)

	pemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}

	manager, err := token_manager.NewTokenManager(pemBytes, []byte{}, logger)
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
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
