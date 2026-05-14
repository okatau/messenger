package components

import (
	"context"
	"fmt"
	"friends_service/internal/repository"
	"friends_service/internal/service"
	"friends_service/pkg/config"
	"friends_service/pkg/service_logger"
	"log"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Env          string                `yaml:"env" env-default:"local"`
	Postgres     config.PostgresConfig `env-prefix:"PG_"`
	ServerConfig config.HTTPConfig     `yaml:"http"`
}

type Components struct {
	Svc      service.Friendship
	Logger   *slog.Logger
	Postgres *pgxpool.Pool
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

	logger := service_logger.InitLogger(cfg.Env)

	userRepo := repository.NewUserRepository(pool)
	friendshipRepo := repository.NewFriendshipRepository(pool)
	svc := service.NewFriendshipService(userRepo, friendshipRepo, logger)

	return &Components{
		Svc:      svc,
		Logger:   logger,
		Postgres: pool,
	}
}

func (c *Components) Shutdown(ctx context.Context) {
	c.Postgres.Close()
}

func getPostgresDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
