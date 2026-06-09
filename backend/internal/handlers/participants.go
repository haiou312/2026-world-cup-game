package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"worldcup/internal/domain"
)

// participants returns every participant (assigned or not) with their team and
// tournament status. Public — anyone can browse and search by name.
func (h *Handler) participants(w http.ResponseWriter, r *http.Request) {
	d, err := h.loadProgress(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	rows, err := h.q.ListParticipants(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	out := make([]map[string]any, 0, len(rows))
	assigned, remaining := 0, 0
	for _, p := range rows {
		m := map[string]any{"id": p.ID, "name": p.Name, "assigned": p.TeamID.Valid}
		if p.TeamID.Valid {
			assigned++
			ts := d.status[p.TeamID.Int64]
			eliminated, champion, stage := false, false, domain.StageGroup
			if ts != nil {
				eliminated, champion, stage = ts.Eliminated, ts.Champion, ts.FurthestStage
			}
			if !eliminated {
				remaining++
			}
			m["team_id"] = p.TeamID.Int64
			m["team_name"] = p.TeamName.String
			m["team_code"] = textVal(p.TeamCode)
			m["flag_url"] = textVal(p.TeamFlagUrl)
			m["group_label"] = textVal(p.TeamGroupLabel)
			m["round"] = p.RoundNumber.Int32
			m["furthest_stage"] = stage
			m["eliminated"] = eliminated
			m["champion"] = champion
		}
		out = append(out, m)
	}

	// Order: assigned-and-alive (deepest stage) → assigned-and-out → unassigned, name tiebreak.
	rank := func(m map[string]any) int {
		if !m["assigned"].(bool) {
			return -2
		}
		if m["eliminated"].(bool) {
			return -1
		}
		return domain.StageRank(m["furthest_stage"].(string))
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := rank(out[i]), rank(out[j])
		if ri != rj {
			return ri > rj
		}
		return out[i]["name"].(string) < out[j]["name"].(string)
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"total":        len(out),
		"assigned":     assigned,
		"unassigned":   len(out) - assigned,
		"remaining":    remaining,
		"participants": out,
	})
}

func (h *Handler) addParticipant(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "name required")
		return
	}
	p, err := h.q.CreateParticipant(r.Context(), strings.TrimSpace(body.Name))
	if err != nil {
		writeError(w, http.StatusConflict, "name already taken")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"id": p.ID, "name": p.Name})
}

func (h *Handler) deleteParticipant(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad id")
		return
	}
	if err := h.q.DeleteParticipant(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, "delete failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// resetAssignments clears every draw so a new round can start from scratch.
func (h *Handler) resetAssignments(w http.ResponseWriter, r *http.Request) {
	if err := h.q.ClearAssignments(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "reset failed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// verifySettings just confirms the password (the middleware already checked it).
func (h *Handler) verifySettings(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
