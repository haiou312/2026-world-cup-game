package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"worldcup/internal/config"
	"worldcup/internal/db"
	"worldcup/internal/handlers"
	"worldcup/internal/sync"
)

func main() {
	_ = godotenv.Load() // load .env for local dev; no-op in docker
	cfg := config.Load()

	if err := db.Migrate(cfg.DatabaseURL); err != nil {
		slog.Error("migrate failed", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect failed", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	if cfg.SettingsPassword == "" {
		slog.Warn("SETTINGS_PASSWORD not set; the Settings section will be inaccessible")
	}

	// Teams are sourced authoritatively from football-data.org on first sync.
	syncer := sync.NewSyncer(pool, cfg.Competition, cfg.MaxAPICallsPerDay)
	if _, err := syncer.Start(ctx); err != nil {
		slog.Error("scheduler start failed", "err", err)
		os.Exit(1)
	}

	h := handlers.New(pool, syncer, cfg.SettingsPassword)

	slog.Info("listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, h.Router()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
