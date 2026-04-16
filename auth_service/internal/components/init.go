package components

import (
	"auth_service/internal/repository"
	"auth_service/internal/service"
	"auth_service/pkg/config"
	"auth_service/pkg/logger"
	"auth_service/pkg/token_manager"
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Env      string                `env:"ENV" env-default:"local"`
	Host     string                `env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port     int                   `env:"SERVER_AUTH_PORT" env-default:"8081"`
	Postgres config.PostgresConfig `env-prefix:"PG_"`
	Auth     AuthConfig            `env-prefix:"AUTH_"`
}

type AuthConfig struct {
	PublicKeyPEMBase64  string `env:"PUBLIC_PEM_BASE64" env-default:""`
	PrivateKeyPEMBase64 string `env:"PRIVATE_PEM_BASE64" env-default:""`
}

type Components struct {
	Postgres     *pgxpool.Pool
	TokenManager *token_manager.TokenManager
	Auth         service.Auth
	Logger       *slog.Logger
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

	publicPemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding public pem")
	}
	privatePemBytes, err := base64.StdEncoding.DecodeString(cfg.Auth.PrivateKeyPEMBase64)
	if err != nil {
		log.Fatal("error decoding private pem")
	}

	manager, err := token_manager.NewTokenManager(publicPemBytes, privatePemBytes, logger)
	if err != nil {
		log.Fatal(err)
	}

	authRepo := repository.NewUserRepositoryPG(pool)
	tokenRepo := repository.NewSessionRepositoryPG(pool)
	auth := service.NewAuth(authRepo, tokenRepo, manager, logger)

	return &Components{
		Postgres:     pool,
		Auth:         auth,
		TokenManager: manager,
		Logger:       logger,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
