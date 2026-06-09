//go:build integration

// Package testsupport spins up throwaway Postgres containers for integration
// tests. Build-tagged "integration" so plain `go test ./...` stays Docker-free.
package testsupport

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"worldcup/internal/db"
)

// NewPostgres starts a fresh Postgres container, applies all migrations, and
// returns a connected pool. The container + pool are cleaned up automatically.
func NewPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	pg, err := postgres.Run(ctx, "postgres:16-alpine",
		postgres.WithDatabase("worldcup_test"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}
	t.Cleanup(func() {
		if err := pg.Terminate(ctx); err != nil {
			t.Logf("terminate container: %v", err)
		}
	})

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("connection string: %v", err)
	}
	if err := db.Migrate(dsn); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// SeedTeams inserts n teams and returns their ids.
func SeedTeams(t *testing.T, pool *pgxpool.Pool, n int) []int64 {
	t.Helper()
	ids := make([]int64, 0, n)
	for i := 0; i < n; i++ {
		var id int64
		if err := pool.QueryRow(context.Background(),
			`INSERT INTO teams (name, code) VALUES ($1, $2) RETURNING id`,
			fmt.Sprintf("Team %02d", i), fmt.Sprintf("T%02d", i),
		).Scan(&id); err != nil {
			t.Fatalf("seed team %d: %v", i, err)
		}
		ids = append(ids, id)
	}
	return ids
}

// CreateParticipant inserts a participant and returns its id.
func CreateParticipant(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO participants (name) VALUES ($1) RETURNING id`,
		name,
	).Scan(&id); err != nil {
		t.Fatalf("create participant %s: %v", name, err)
	}
	return id
}
