package components

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"auth_service/internal/repository"
	"auth_service/internal/service"
	"auth_service/pkg/config"
	"auth_service/pkg/service_logger"
	"auth_service/pkg/token_manager"
)

type Config struct {
	Env          string                `env:"ENV" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	Auth         config.AuthConfig
	ServerConfig config.ServerConfig `yaml:"http"`
}

type Components struct {
	Postgres     *pgxpool.Pool
	TokenManager *token_manager.TokenManager
	Svc          service.Auth
	Logger       *slog.Logger
}

func InitComponents(ctx context.Context, cfg *Config) *Components {
	logger := service_logger.InitLogger(cfg.Env)

	pool := initPG(ctx, cfg.Postgres)
	manager := initTokenManager(cfg.Auth, logger)
	svc := initSvc(pool, manager, logger, cfg.Auth.RefreshTokenTTL)

	return &Components{
		Postgres:     pool,
		Svc:          svc,
		TokenManager: manager,
		Logger:       logger,
	}
}

func (c *Components) Shutdown() {
	c.Postgres.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}

func initPG(ctx context.Context, cfg config.PostgresConfig) *pgxpool.Pool {
	dsn := getPostgresDSN(cfg)

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatal(err)
	}
	if err = pool.Ping(ctx); err != nil {
		log.Fatal(err)
	}

	return pool
}

func initTokenManager(cfg config.AuthConfig, logger *slog.Logger) *token_manager.TokenManager {
	publicPemBytes, err := base64.StdEncoding.DecodeString(cfg.PublicKeyPEMBase64)
	if err != nil {
		log.Fatal("invalid public pem")
	}
	privatePemBytes, err := base64.StdEncoding.DecodeString(cfg.PrivateKeyPEMBase64)
	if err != nil {
		log.Fatal("invalid private pem")
	}

	manager, err := token_manager.NewTokenManager(publicPemBytes, privatePemBytes, cfg.AccessTokenTTL, logger)
	if err != nil {
		log.Fatal(err)
	}

	return manager
}

func initSvc(
	pool *pgxpool.Pool,
	manager *token_manager.TokenManager,
	logger *slog.Logger,
	refreshTokenTTL time.Duration,
) service.Auth {
	authRepo := repository.NewUserRepository(pool)
	tokenRepo := repository.NewSessionRepository(pool)
	return service.New(authRepo, tokenRepo, manager, logger, refreshTokenTTL)
}
