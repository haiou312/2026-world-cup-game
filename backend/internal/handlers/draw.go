package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
)

// assign is the "spin the country wheel" action: it gives the chosen participant
// a random still-available team (advisory-locked, round-aware). Settings-gated.
func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ParticipantID int64 `json:"participant_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ParticipantID == 0 {
		writeError(w, http.StatusBadRequest, "participant_id required")
		return
	}
	a, err := domain.Draw(r.Context(), h.pool, h.q, body.ParticipantID)
	if err != nil {
		if errors.Is(err, domain.ErrNoTeams) {
			writeError(w, http.StatusServiceUnavailable, "no teams available yet")
			return
		}
		writeError(w, http.StatusInternalServerError, "assign failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assignment": assignmentView(a)})
}

func (h *Handler) teams(w http.ResponseWriter, r *http.Request) {
	ts, err := h.q.ListTeams(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	out := make([]map[string]any, 0, len(ts))
	for _, t := range ts {
		out = append(out, map[string]any{
			"id":          t.ID,
			"name":        t.Name,
			"code":        textVal(t.Code),
			"flag_url":    textVal(t.FlagUrl),
			"group_label": textVal(t.GroupLabel),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"teams": out})
}

func assignmentView(a sqlc.GetAssignmentWithTeamRow) map[string]any {
	return map[string]any{
		"participant_id": a.ParticipantID,
		"team_id":        a.TeamID,
		"team_name":      a.TeamName,
		"team_code":      textVal(a.TeamCode),
		"flag_url":       textVal(a.TeamFlagUrl),
		"group_label":    textVal(a.TeamGroupLabel),
		"round":          a.RoundNumber,
	}
}
