package sync

import (
	"context"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
)

const tokenSetting = "football_data_token"

type Syncer struct {
	pool        *pgxpool.Pool
	q           *sqlc.Queries
	competition string
	maxCalls    int
	running     atomic.Bool
}

func NewSyncer(pool *pgxpool.Pool, competition string, maxCalls int) *Syncer {
	return &Syncer{pool: pool, q: sqlc.New(pool), competition: competition, maxCalls: maxCalls}
}

// APIKey returns the configured football-data.org token, or "" if unset.
func (s *Syncer) APIKey(ctx context.Context) string {
	v, err := s.q.GetSetting(ctx, tokenSetting)
	if err != nil || !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}

// SyncOnce performs one sync pass. scope "daily" fetches only today's matches;
// scope "full" fetches the whole competition (seed / manual catch-up).
func (s *Syncer) SyncOnce(ctx context.Context, scope string) error {
	// Only one sync at a time — prevents a manual sync from colliding with the
	// scheduled / token-save background sync and producing partial data.
	if !s.running.CompareAndSwap(false, true) {
		slog.Info("sync skipped: another sync is already running")
		return nil
	}
	defer s.running.Store(false)

	token := s.APIKey(ctx)
	if token == "" {
		slog.Warn("sync skipped: football-data token not configured")
		return nil
	}

	st, err := s.q.GetSyncState(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	todayDate := pgtype.Date{Time: time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC), Valid: true}

	calls := st.ApiCallsToday
	todayDone := st.TodayDone
	if !st.CallsDate.Valid || !st.CallsDate.Time.Equal(todayDate.Time) {
		calls = 0 // new day → reset budget and re-open polling
		todayDone = false
	}
	if scope == "daily" && todayDone {
		slog.Info("sync skipped: today's matches already complete")
		return nil
	}
	if int(calls) >= s.maxCalls {
		slog.Warn("sync skipped: daily api budget reached", "calls", calls, "max", s.maxCalls)
		return nil
	}

	date := ""
	if scope == "daily" {
		date = now.Format("2006-01-02")
	}

	client := NewClient(token)
	matches, err := client.Matches(ctx, s.competition, date)
	calls++ // count the call regardless of outcome
	if err != nil {
		_ = s.persistState(ctx, calls, todayDate, todayDone)
		_ = s.q.RecordSyncError(ctx, sqlc.RecordSyncErrorParams{
			LastError:   err.Error(),
			LastErrorAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		})
		return err
	}

	cache, err := s.buildTeamCache(ctx)
	if err != nil {
		return err
	}

	allFinished := true
	for _, m := range matches {
		if err := s.upsertOne(ctx, cache, m); err != nil {
			slog.Error("upsert match failed", "match_id", m.ID, "err", err)
			continue
		}
		if !domain.IsFinished(fdStatus(m.Status, m.Score.Duration)) {
			allFinished = false
		}
	}

	if scope == "daily" {
		todayDone = len(matches) == 0 || allFinished
	}

	// Group standings (best-effort, +1 API call). Already sorted by the official
	// tie-breakers via each row's position.
	standings, sErr := client.Standings(ctx, s.competition)
	calls++
	if sErr != nil {
		slog.Error("standings fetch failed", "err", sErr)
	} else {
		s.upsertStandings(ctx, cache, standings)
	}

	if err := s.persistState(ctx, calls, todayDate, todayDone); err != nil {
		return err
	}

	// Record a clean success + any non-fatal warnings (unmatched teams, a failed
	// standings fetch) so the admin can see sync health instead of digging logs.
	var warnings []string
	if u := dedupeStrings(cache.unmatched); len(u) > 0 {
		warnings = append(warnings, "unmatched teams: "+strings.Join(u, ", "))
	}
	if sErr != nil {
		warnings = append(warnings, "standings fetch failed: "+sErr.Error())
	}
	_ = s.q.RecordSyncSuccess(ctx, sqlc.RecordSyncSuccessParams{
		LastSuccessAt: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		LastWarnings:  strings.Join(warnings, " | "),
	})

	slog.Info("sync done", "scope", scope, "matches", len(matches), "calls_today", calls, "today_done", todayDone, "warnings", len(warnings))
	return nil
}

func dedupeStrings(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

func (s *Syncer) upsertStandings(ctx context.Context, cache *teamCache, groups []fdStanding) {
	for _, g := range groups {
		grp := strings.TrimPrefix(g.Group, "Group ")
		for _, row := range g.Table {
			teamID, err := s.resolveTeam(ctx, cache, row.Team)
			if err != nil || teamID == 0 {
				continue
			}
			if err := s.q.UpsertStanding(ctx, sqlc.UpsertStandingParams{
				TeamID:       teamID,
				GroupLabel:   pgtype.Text{String: grp, Valid: grp != ""},
				Position:     int32(row.Position),
				Played:       int32(row.PlayedGames),
				Won:          int32(row.Won),
				Draw:         int32(row.Draw),
				Lost:         int32(row.Lost),
				GoalsFor:     int32(row.GoalsFor),
				GoalsAgainst: int32(row.GoalsAgainst),
				GoalDiff:     int32(row.GoalDifference),
				Points:       int32(row.Points),
			}); err != nil {
				slog.Error("upsert standing failed", "team", row.Team.Name, "err", err)
			}
		}
	}
}

func (s *Syncer) persistState(ctx context.Context, calls int32, date pgtype.Date, todayDone bool) error {
	return s.q.UpdateSyncState(ctx, sqlc.UpdateSyncStateParams{
		LastSyncedAt:  pgtype.Timestamptz{Time: time.Now(), Valid: true},
		TodayDone:     todayDone,
		ApiCallsToday: calls,
		CallsDate:     date,
	})
}

func (s *Syncer) upsertOne(ctx context.Context, cache *teamCache, m fdMatch) error {
	homeID, err := s.resolveTeam(ctx, cache, m.HomeTeam)
	if err != nil {
		return err
	}
	awayID, err := s.resolveTeam(ctx, cache, m.AwayTeam)
	if err != nil {
		return err
	}

	stage := fdStageToStage(m.Stage)

	group := pgtype.Text{}
	if strings.HasPrefix(m.Group, "GROUP_") {
		group = pgtype.Text{String: strings.TrimPrefix(m.Group, "GROUP_"), Valid: true}
	}

	// football-data is authoritative for group membership: write the match's
	// group onto the team rows so the group badges stay correct.
	if stage == domain.StageGroup && group.Valid {
		if err := s.setTeamGroup(ctx, cache, homeID, group.String); err != nil {
			return err
		}
		if err := s.setTeamGroup(ctx, cache, awayID, group.String); err != nil {
			return err
		}
	}

	var winner int64
	switch m.Score.Winner {
	case "HOME_TEAM":
		winner = homeID
	case "AWAY_TEAM":
		winner = awayID
	}

	return s.q.UpsertFixture(ctx, sqlc.UpsertFixtureParams{
		ApiFixtureID: int32(m.ID),
		Stage:        stage,
		RoundLabel:   pgtype.Text{String: m.Stage, Valid: m.Stage != ""},
		GroupLabel:   group,
		HomeTeamID:   int8OrNull(homeID),
		AwayTeamID:   int8OrNull(awayID),
		HomeScore:    int4Ptr(m.Score.FullTime.Home),
		AwayScore:    int4Ptr(m.Score.FullTime.Away),
		Status:       fdStatus(m.Status, m.Score.Duration),
		WinnerTeamID: int8OrNull(winner),
		KickoffAt:    pgtype.Timestamptz{Time: m.UtcDate, Valid: !m.UtcDate.IsZero()},
	})
}

func fdStageToStage(stage string) string {
	switch stage {
	case "GROUP_STAGE":
		return domain.StageGroup
	case "LAST_32":
		return domain.StageR32
	case "LAST_16":
		return domain.StageR16
	case "QUARTER_FINALS":
		return domain.StageQF
	case "SEMI_FINALS":
		return domain.StageSF
	case "THIRD_PLACE", "THIRD_PLACE_FINAL":
		return domain.StageThird
	case "FINAL":
		return domain.StageFinal
	default:
		return domain.StageGroup
	}
}

// fdStatus maps football-data status (+ score duration) to our internal codes
// understood by domain.IsFinished / domain.IsLive.
func fdStatus(status, duration string) string {
	switch status {
	case "FINISHED", "AWARDED":
		switch duration {
		case "PENALTY_SHOOTOUT":
			return "PEN"
		case "EXTRA_TIME":
			return "AET"
		default:
			return "FT"
		}
	case "IN_PLAY":
		return "LIVE"
	case "PAUSED":
		return "HT"
	case "SUSPENDED":
		return "SUSP"
	default: // SCHEDULED, TIMED, POSTPONED, CANCELLED
		return "NS"
	}
}

func int8OrNull(id int64) pgtype.Int8 {
	if id == 0 {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: id, Valid: true}
}

func int4Ptr(v *int) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: int32(*v), Valid: true}
}
