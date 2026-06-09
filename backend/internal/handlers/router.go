package handlers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/sync"
)

type Handler struct {
	q                *sqlc.Queries
	pool             *pgxpool.Pool
	syncer           *sync.Syncer
	settingsPassword string
}

func New(pool *pgxpool.Pool, syncer *sync.Syncer, settingsPassword string) *Handler {
	return &Handler{q: sqlc.New(pool), pool: pool, syncer: syncer, settingsPassword: settingsPassword}
}

// requireSettings gates the host-only Settings/Wheel actions behind the shared
// SETTINGS_PASSWORD (sent as the X-Settings-Password header).
func (h *Handler) requireSettings(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pw := r.Header.Get("X-Settings-Password")
		if h.settingsPassword == "" || pw != h.settingsPassword {
			writeError(w, http.StatusUnauthorized, "invalid settings password")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Content-Type", "X-Settings-Password"},
	}))

	// Public reads — no account needed.
	r.Get("/api/health", h.health)
	r.Get("/api/teams", h.teams)
	r.Get("/api/bracket", h.bracket)
	r.Get("/api/participants", h.participants)
	r.Get("/api/fixtures", h.fixtures)

	// Host-only actions, gated by the settings password.
	r.Group(func(r chi.Router) {
		r.Use(h.requireSettings)
		r.Post("/api/settings/verify", h.verifySettings)
		r.Post("/api/participants", h.addParticipant)
		r.Delete("/api/participants/{id}", h.deleteParticipant)
		r.Post("/api/assign", h.assign)
		r.Post("/api/reset", h.resetAssignments)
		r.Get("/api/settings/sync-status", h.adminSyncStatus)
		r.Put("/api/settings/api-key", h.adminSetAPIKey)
		r.Post("/api/settings/sync", h.adminSync)
	})

	return r
}
