//go:build integration

package domain_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
	"worldcup/internal/testsupport"
)

// TestDrawConcurrencyAndRoundReset hammers Draw with 48 simultaneous draws and
// asserts the advisory lock keeps every team unique within a round, that the
// pool resets cleanly for the next round, and that re-drawing is idempotent.
func TestDrawConcurrencyAndRoundReset(t *testing.T) {
	pool := testsupport.NewPostgres(t)
	q := sqlc.New(pool)
	ctx := context.Background()

	teamIDs := testsupport.SeedTeams(t, pool, 48)
	teamSet := map[int64]bool{}
	for _, id := range teamIDs {
		teamSet[id] = true
	}

	const total = 96 // two full rounds
	users := make([]int64, total)
	for i := range users {
		users[i] = testsupport.CreateParticipant(t, pool, fmt.Sprintf("p%03d", i))
	}

	drawRound := func(batch []int64, wantRound int32) {
		var mu sync.Mutex
		got := map[int64]int{}
		var wg sync.WaitGroup
		errs := make(chan error, len(batch))
		for _, uid := range batch {
			wg.Add(1)
			go func(uid int64) {
				defer wg.Done()
				a, err := domain.Draw(ctx, pool, q, uid)
				if err != nil {
					errs <- fmt.Errorf("user %d: %w", uid, err)
					return
				}
				if a.RoundNumber != wantRound {
					errs <- fmt.Errorf("user %d got round %d, want %d", uid, a.RoundNumber, wantRound)
					return
				}
				mu.Lock()
				got[a.TeamID]++
				mu.Unlock()
			}(uid)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Error(err)
		}
		if len(got) != 48 {
			t.Errorf("round %d: %d distinct teams, want 48", wantRound, len(got))
		}
		for id, c := range got {
			if c != 1 {
				t.Errorf("round %d: team %d assigned %d times, want 1", wantRound, id, c)
			}
			if !teamSet[id] {
				t.Errorf("round %d: unexpected team %d", wantRound, id)
			}
		}
	}

	// Round 0: 48 concurrent draws → 48 unique teams.
	drawRound(users[:48], 0)
	// Round 1: next 48 concurrent draws → pool resets, 48 unique teams again.
	drawRound(users[48:], 1)

	// Idempotency: the same user re-drawing returns the same team and round.
	first, err := domain.Draw(ctx, pool, q, users[0])
	if err != nil {
		t.Fatalf("idempotent draw: %v", err)
	}
	again, err := domain.Draw(ctx, pool, q, users[0])
	if err != nil {
		t.Fatalf("idempotent redraw: %v", err)
	}
	if first.TeamID != again.TeamID || again.RoundNumber != 0 {
		t.Errorf("idempotency broken: first=%d/%d again=%d/%d",
			first.TeamID, first.RoundNumber, again.TeamID, again.RoundNumber)
	}
}
