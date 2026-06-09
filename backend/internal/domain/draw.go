package domain

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/db/sqlc"
)

const teamsPerRound = 48

var ErrNoTeams = errors.New("no teams available to draw")

// Draw assigns a random team to a participant. Within each round of 48 draws
// every team is unique; after 48 the pool resets for the next round. Idempotent:
// a participant who already drew gets their existing team back.
func Draw(ctx context.Context, pool *pgxpool.Pool, q *sqlc.Queries, participantID int64) (sqlc.GetAssignmentWithTeamRow, error) {
	var empty sqlc.GetAssignmentWithTeamRow

	tx, err := pool.Begin(ctx)
	if err != nil {
		return empty, err
	}
	defer tx.Rollback(ctx)
	qtx := q.WithTx(tx)

	// Serialize all draws so the count→pick→insert sequence is atomic.
	if err := qtx.AcquireDrawLock(ctx); err != nil {
		return empty, err
	}

	// Idempotent: already drawn?
	if existing, err := qtx.GetAssignmentWithTeam(ctx, participantID); err == nil {
		return existing, nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return empty, err
	}

	total, err := qtx.CountAssignments(ctx)
	if err != nil {
		return empty, err
	}
	round := int32(total / teamsPerRound)

	teamID, err := qtx.PickAvailableTeam(ctx, round)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return empty, ErrNoTeams
		}
		return empty, err
	}

	if _, err := qtx.CreateAssignment(ctx, sqlc.CreateAssignmentParams{
		ParticipantID: participantID,
		TeamID:        teamID,
		RoundNumber:   round,
	}); err != nil {
		return empty, err
	}

	result, err := qtx.GetAssignmentWithTeam(ctx, participantID)
	if err != nil {
		return empty, err
	}
	if err := tx.Commit(ctx); err != nil {
		return empty, err
	}
	return result, nil
}
