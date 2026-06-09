package db

import (
	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"worldcup/migrations"
)

// Migrate applies all pending goose migrations using the embedded SQL files.
func Migrate(databaseURL string) error {
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		return err
	}

	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	return goose.Up(sqlDB, ".")
}
