package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"worldcup/internal/db/sqlc"
)

const apiKeySetting = "football_data_token"

func (h *Handler) apiKeyConfigured(r *http.Request) (bool, string) {
	v, err := h.q.GetSetting(r.Context(), apiKeySetting)
	if err != nil || !v.Valid || v.String == "" {
		return false, ""
	}
	return true, maskKey(v.String)
}

func (h *Handler) adminSetAPIKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Key) == "" {
		writeError(w, http.StatusBadRequest, "key required")
		return
	}
	if err := h.q.UpsertSetting(r.Context(), sqlc.UpsertSettingParams{
		Key:       apiKeySetting,
		Value:     pgtype.Text{String: strings.TrimSpace(body.Key), Valid: true},
		UpdatedBy: pgtype.Int8{},
	}); err != nil {
		writeError(w, http.StatusInternalServerError, "save failed")
		return
	}
	// Seed teams/fixtures in the background now that a key exists.
	if h.syncer != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
			defer cancel()
			_ = h.syncer.SyncOnce(ctx, "full")
		}()
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) adminSyncStatus(w http.ResponseWriter, r *http.Request) {
	st, err := h.q.GetSyncState(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	configured, masked := h.apiKeyConfigured(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"api_key_configured": configured,
		"api_key_masked":     masked,
		"last_synced_at":     tsVal(st.LastSyncedAt),
		"last_success_at":    tsVal(st.LastSuccessAt),
		"last_error":         st.LastError,
		"last_error_at":      tsVal(st.LastErrorAt),
		"last_warnings":      st.LastWarnings,
		"today_done":         st.TodayDone,
		"api_calls_today":    st.ApiCallsToday,
		"calls_date":         dateVal(st.CallsDate),
	})
}

func (h *Handler) adminSync(w http.ResponseWriter, r *http.Request) {
	if h.syncer == nil {
		writeError(w, http.StatusServiceUnavailable, "sync unavailable")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err := h.syncer.SyncOnce(ctx, "full"); err != nil {
		writeError(w, http.StatusBadGateway, "sync failed: "+err.Error())
		return
	}
	h.adminSyncStatus(w, r)
}
