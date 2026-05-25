package db

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"

	"chat_service/pkg/config"
)

func Connect(ctx context.Context, pgcfg config.PostgresConfig) *pgxpool.Pool {
	dsn := getPostgresDSN(pgcfg)

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

func getMigrateDSN(cfg config.PostgresConfig) string {
	return fmt.Sprintf("pgx5://%s:%s@%s:%d/%s", cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)
}
