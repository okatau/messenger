package components

import (
	"context"
	"fmt"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	"friends_service/internal/repository"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
)

type Config struct {
	Env          string                `yaml:"env" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	ServerConfig config.ServerConfig   `yaml:"http"`
}

type Components struct {
	Svc      service.Friendship
	Logger   *slog.Logger
	Postgres *pgxpool.Pool
}

func InitComponents(ctx context.Context, cfg *Config) *Components {
	logger := service_logger.InitLogger(cfg.Env)

	pool := initPG(ctx, cfg.Postgres)
	svc := initSvc(pool, logger)

	return &Components{
		Svc:      svc,
		Logger:   logger,
		Postgres: pool,
	}
}

func (c *Components) Shutdown() {
	c.Postgres.Close()
}

func initSvc(
	pool *pgxpool.Pool,
	logger *slog.Logger,
) service.Friendship {
	userRepo := repository.NewUserRepository(pool)
	friendshipRepo := repository.NewFriendshipRepository(pool)
	return service.NewFriendshipService(userRepo, friendshipRepo, logger)
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

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
