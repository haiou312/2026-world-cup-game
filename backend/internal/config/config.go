package config

import (
	"os"
	"strconv"
)

type Config struct {
	DatabaseURL      string
	SettingsPassword string
	Port             string

	Competition       string
	MaxAPICallsPerDay int
}

func Load() Config {
	return Config{
		DatabaseURL:       getenv("DATABASE_URL", "postgres://wc:change_me_pg@localhost:6573/worldcup?sslmode=disable"),
		SettingsPassword:  getenv("SETTINGS_PASSWORD", ""),
		Port:              getenv("PORT", "8080"),
		Competition:       getenv("FD_COMPETITION", "WC"),
		MaxAPICallsPerDay: getenvInt("SYNC_MAX_CALLS_PER_DAY", 80),
	}
}

func getenvInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
