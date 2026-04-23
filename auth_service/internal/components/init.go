package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"time"

	"auth_service/internal/repository"
	"auth_service/internal/service"
	"auth_service/pkg/config"
	"auth_service/pkg/logger"
	"auth_service/pkg/token_manager"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Env          string                `env:"ENV" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	Auth         AuthConfig
	ServerConfig config.HTTPConfig `yaml:"http"`
}

type AuthConfig struct {
	AccessTokenTTL      time.Duration `yaml:"access_token_ttl" env-default:"15m"`
	RefreshTokenTTL     time.Duration `yaml:"refresh_token_ttl" env-default:"720h"` // 30 days
	PublicKeyPEMBase64  string        `env:"AUTH_PUBLIC_PEM_BASE64" env-required:"true"`
	PrivateKeyPEMBase64 string        `env:"AUTH_PRIVATE_PEM_BASE64" env-required:"true"`
}

type Components struct {
	Postgres     *pgxpool.Pool
	TokenManager *token_manager.TokenManager
	Svc          service.Auth
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

	manager, err := token_manager.NewTokenManager(publicPemBytes, privatePemBytes, cfg.Auth.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	authRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewSessionRepository(pool)
	svc := service.NewAuthService(authRepo, tokenRepo, manager, logger, cfg.Auth.RefreshTokenTTL)

	return &Components{
		Postgres:     pool,
		Svc:          svc,
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
