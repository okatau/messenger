package db

import (
	"embed"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5" // import pgx driver
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"chat_service/pkg/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func Run(pgcfg config.PostgresConfig) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return err
	}
	dsn := getMigrateDSN(pgcfg)
	m, err := migrate.NewWithSourceInstance("iofs", src, dsn)
	if err != nil {
		return err
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			log.Printf("migrate source close error: %v", srcErr)
		}
		if dbErr != nil {
			log.Printf("migrate db close error: %v", dbErr)
		}
	}()

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	return nil
}
