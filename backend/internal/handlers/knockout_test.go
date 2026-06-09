package handlers

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
)

func mkInt8(v int64) pgtype.Int8 {
	if v == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: v, Valid: true}
}

func koRow(id int64, stage string, home, away, winner int64) sqlc.ListFixturesWithTeamsRow {
	return sqlc.ListFixturesWithTeamsRow{
		ID:           id,
		Stage:        stage,
		Status:       "FT",
		HomeTeamID:   mkInt8(home),
		AwayTeamID:   mkInt8(away),
		WinnerTeamID: mkInt8(winner),
	}
}

func roundFixtures(rounds []map[string]any, stage string) []map[string]any {
	for _, r := range rounds {
		if r["stage"] == stage {
			return r["fixtures"].([]map[string]any)
		}
	}
	return nil
}

func sideID(slot map[string]any, side string) int64 {
	if s, ok := slot[side].(map[string]any); ok {
		if id, ok := s["team_id"].(int64); ok {
			return id
		}
	}
	return 0
}

// TestResolveKnockoutNonScramble feeds resolveKnockout a seeded post-group
// scenario and asserts the bracket resolves correctly: R32 top seeds come from
// the standings, the 3rd-place opponent is read off the real fixture, and the
// R16 slot is exactly the two feeding winners — with the upper feeder on top
// even though the real R16 fixture stores them home/away-reversed (the
// anti-scramble guarantee, which was previously only verified by the simulator).
func TestResolveKnockoutNonScramble(t *testing.T) {
	const (
		e1  = int64(51) // group E winner → seed "1E"
		i1  = int64(91) // group I winner → seed "1I"
		t3a = int64(60) // 3rd-place opponent in R32 slot 0
		t3b = int64(61) // 3rd-place opponent in R32 slot 1
	)
	team := func(id int64, name string) sqlc.Team { return sqlc.Team{ID: id, Name: name} }

	d := &progressData{
		players:       map[int64][]string{},
		status:        map[int64]*domain.TeamStatus{},
		bestThird:     map[int64]bool{},
		groupComplete: map[string]bool{"E": true, "I": true},
		standings: map[int64]sqlc.Standing{
			e1: {TeamID: e1, GroupLabel: pgtype.Text{String: "E", Valid: true}, Position: 1},
			i1: {TeamID: i1, GroupLabel: pgtype.Text{String: "I", Valid: true}, Position: 1},
		},
		teamByID: map[int64]sqlc.Team{
			e1:  team(e1, "Group E winner"),
			i1:  team(i1, "Group I winner"),
			t3a: team(t3a, "Third A"),
			t3b: team(t3b, "Third B"),
		},
		fixtures: []sqlc.ListFixturesWithTeamsRow{
			koRow(1, domain.StageR32, e1, t3a, e1), // 1E beats a 3rd-place team
			koRow(2, domain.StageR32, i1, t3b, i1), // 1I beats a 3rd-place team
			koRow(3, domain.StageR16, i1, e1, 0),   // R16 winners meet — stored REVERSED
		},
	}

	rounds := (&Handler{}).resolveKnockout(d)

	r32 := roundFixtures(rounds, domain.StageR32)
	if len(r32) != 16 {
		t.Fatalf("R32 fixtures = %d, want 16", len(r32))
	}
	if got := sideID(r32[0], "home"); got != e1 {
		t.Errorf("R32[0] home = %d, want %d (1E from standings)", got, e1)
	}
	if got := sideID(r32[0], "away"); got != t3a {
		t.Errorf("R32[0] away = %d, want %d (3rd read from fixture)", got, t3a)
	}
	if got := sideID(r32[1], "home"); got != i1 {
		t.Errorf("R32[1] home = %d, want %d (1I)", got, i1)
	}
	if got := sideID(r32[1], "away"); got != t3b {
		t.Errorf("R32[1] away = %d, want %d (3rd)", got, t3b)
	}

	r16 := roundFixtures(rounds, domain.StageR16)
	if len(r16) != 8 {
		t.Fatalf("R16 fixtures = %d, want 8", len(r16))
	}
	// Must be exactly the two feeding winners, upper feeder (e1) on top despite
	// the fixture being stored as (home=i1, away=e1).
	if got := sideID(r16[0], "home"); got != e1 {
		t.Errorf("R16[0] home = %d, want %d (winner of R32[0]); scrambled?", got, e1)
	}
	if got := sideID(r16[0], "away"); got != i1 {
		t.Errorf("R16[0] away = %d, want %d (winner of R32[1]); scrambled?", got, i1)
	}

	// Untouched slots stay TBD (no team_id), proving we don't invent matchups.
	if got := sideID(r16[1], "home"); got != 0 {
		t.Errorf("R16[1] home = %d, want 0 (TBD)", got)
	}
}
