//go:build integration

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/testsupport"
)

// TestBracketStructure drives the real /api/bracket handler against a seeded DB
// and asserts the skeleton renders 12 groups and the full knockout tree with the
// correct per-stage fixture counts, all pending pre-tournament.
func TestBracketStructure(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	ctx := context.Background()

	const groups = "ABCDEFGHIJKL"
	for _, g := range groups {
		for i := 0; i < 4; i++ {
			if _, err := pool.Exec(ctx,
				`INSERT INTO teams (name, group_label) VALUES ($1, $2)`,
				fmt.Sprintf("Team %c%d", g, i), string(g),
			); err != nil {
				t.Fatalf("seed team: %v", err)
			}
		}
	}

	h := &Handler{q: sqlc.New(pool), pool: pool}
	req := httptest.NewRequest(http.MethodGet, "/api/bracket", nil)
	rec := httptest.NewRecorder()
	h.bracket(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Groups []struct {
			Group string `json:"group"`
			Teams []struct {
				AdvStatus string `json:"adv_status"`
			} `json:"teams"`
		} `json:"groups"`
		Rounds []struct {
			Stage    string            `json:"stage"`
			Fixtures []json.RawMessage `json:"fixtures"`
		} `json:"rounds"`
		Champion json.RawMessage `json:"champion"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(resp.Groups) != 12 {
		t.Errorf("groups = %d, want 12", len(resp.Groups))
	}
	for _, g := range resp.Groups {
		if len(g.Teams) != 4 {
			t.Errorf("group %s has %d teams, want 4", g.Group, len(g.Teams))
		}
		for _, tm := range g.Teams {
			if tm.AdvStatus != "pending" {
				t.Errorf("group %s adv_status=%s, want pending (nothing played)", g.Group, tm.AdvStatus)
			}
		}
	}

	want := map[string]int{"R32": 16, "R16": 8, "QF": 4, "SF": 2, "THIRD": 1, "FINAL": 1}
	got := map[string]int{}
	for _, r := range resp.Rounds {
		got[r.Stage] = len(r.Fixtures)
	}
	for stage, n := range want {
		if got[stage] != n {
			t.Errorf("stage %s fixtures = %d, want %d", stage, got[stage], n)
		}
	}

	if string(resp.Champion) != "null" {
		t.Errorf("champion = %s, want null", resp.Champion)
	}
}
