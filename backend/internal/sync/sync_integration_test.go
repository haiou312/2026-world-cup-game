//go:build integration

package sync

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/testsupport"
)

func seedTeam(t *testing.T, pool *pgxpool.Pool, name, code string) int64 {
	t.Helper()
	var id int64
	if err := pool.QueryRow(context.Background(),
		`INSERT INTO teams (name, code) VALUES ($1, $2) RETURNING id`, name, code,
	).Scan(&id); err != nil {
		t.Fatalf("seed team %s: %v", name, err)
	}
	return id
}

// TestSyncUpsert exercises the real upsert path: team resolution (by TLA + by
// name alias), group/knockout fixture writes, status/score mapping, group-label
// propagation, and standings upsert — all against a real Postgres.
func TestSyncUpsert(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	ctx := context.Background()

	bra := seedTeam(t, pool, "Brazil", "BRA")
	arg := seedTeam(t, pool, "Argentina", "ARG")
	usa := seedTeam(t, pool, "USA", "")

	s := NewSyncer(pool, "WC", 100)
	cache, err := s.buildTeamCache(ctx)
	if err != nil {
		t.Fatalf("build cache: %v", err)
	}

	// Finished group-stage match, teams resolved by TLA.
	m := fdMatch{ID: 101, Status: "FINISHED", Stage: "GROUP_STAGE", Group: "GROUP_A",
		HomeTeam: fdTeam{ID: 9001, Name: "Brazil", Tla: "BRA"},
		AwayTeam: fdTeam{ID: 9002, Name: "Argentina", Tla: "ARG"}}
	m.Score.Winner = "HOME_TEAM"
	m.Score.Duration = "REGULAR"
	h, a := 2, 1
	m.Score.FullTime.Home, m.Score.FullTime.Away = &h, &a
	if err := s.upsertOne(ctx, cache, m); err != nil {
		t.Fatalf("upsert group match: %v", err)
	}

	var stage, status string
	var hs, as, homeID, awayID, winnerID int64
	if err := pool.QueryRow(ctx,
		`SELECT stage, status, home_score, away_score, home_team_id, away_team_id, winner_team_id
		   FROM fixtures WHERE api_fixture_id = 101`,
	).Scan(&stage, &status, &hs, &as, &homeID, &awayID, &winnerID); err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if stage != "GROUP" || status != "FT" {
		t.Errorf("stage/status = %s/%s, want GROUP/FT", stage, status)
	}
	if homeID != bra || awayID != arg || winnerID != bra {
		t.Errorf("ids home=%d away=%d winner=%d, want %d/%d/%d", homeID, awayID, winnerID, bra, arg, bra)
	}
	if hs != 2 || as != 1 {
		t.Errorf("score = %d-%d, want 2-1", hs, as)
	}

	// Group label propagated onto the team rows.
	var grp string
	if err := pool.QueryRow(ctx, `SELECT group_label FROM teams WHERE id = $1`, bra).Scan(&grp); err != nil {
		t.Fatalf("read group: %v", err)
	}
	if grp != "A" {
		t.Errorf("brazil group = %q, want A", grp)
	}

	// TBD knockout match → null teams, status NS, stage R32.
	if err := s.upsertOne(ctx, cache, fdMatch{ID: 201, Status: "TIMED", Stage: "LAST_32"}); err != nil {
		t.Fatalf("upsert knockout match: %v", err)
	}
	var koStage, koStatus string
	var koHome, koAway *int64
	if err := pool.QueryRow(ctx,
		`SELECT stage, status, home_team_id, away_team_id FROM fixtures WHERE api_fixture_id = 201`,
	).Scan(&koStage, &koStatus, &koHome, &koAway); err != nil {
		t.Fatalf("read ko fixture: %v", err)
	}
	if koStage != "R32" || koStatus != "NS" || koHome != nil || koAway != nil {
		t.Errorf("ko fixture stage=%s status=%s home=%v away=%v, want R32/NS/nil/nil", koStage, koStatus, koHome, koAway)
	}

	// Name-alias matching: provider "United States" → seeded "USA".
	id, err := s.resolveTeam(ctx, cache, fdTeam{ID: 9003, Name: "United States"})
	if err != nil {
		t.Fatalf("resolve alias: %v", err)
	}
	if id != usa {
		t.Errorf("United States resolved to %d, want USA %d", id, usa)
	}

	// Standings upsert writes positions/points.
	s.upsertStandings(ctx, cache, []fdStanding{{Group: "Group A", Table: []fdStandingRow{
		{Position: 1, Team: fdTeam{ID: 9001, Name: "Brazil", Tla: "BRA"}, PlayedGames: 3, Won: 3, Points: 9, GoalsFor: 6, GoalsAgainst: 1, GoalDifference: 5},
		{Position: 2, Team: fdTeam{ID: 9002, Name: "Argentina", Tla: "ARG"}, PlayedGames: 3, Won: 2, Points: 6},
	}}})
	var pos, pts int
	if err := pool.QueryRow(ctx, `SELECT position, points FROM standings WHERE team_id = $1`, bra).Scan(&pos, &pts); err != nil {
		t.Fatalf("read standing: %v", err)
	}
	if pos != 1 || pts != 9 {
		t.Errorf("brazil standing pos=%d pts=%d, want 1/9", pos, pts)
	}
}
