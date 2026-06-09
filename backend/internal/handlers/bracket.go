package handlers

import (
	"context"
	"net/http"
	"sort"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"

	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
)

// progressData bundles everything the read views need in one DB round-trip set.
type progressData struct {
	fixtures     []sqlc.ListFixturesWithTeamsRow
	teams        []sqlc.Team
	players      map[int64][]string
	status       map[int64]*domain.TeamStatus
	standings     map[int64]sqlc.Standing
	bestThird     map[int64]bool  // position-3 teams currently in the best-8
	groupComplete map[string]bool // every team in the group has played 3
	teamByID      map[int64]sqlc.Team
}

func (h *Handler) loadProgress(ctx context.Context) (*progressData, error) {
	fixtures, err := h.q.ListFixturesWithTeams(ctx)
	if err != nil {
		return nil, err
	}
	teams, err := h.q.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	tp, err := h.q.ListTeamPlayers(ctx)
	if err != nil {
		return nil, err
	}

	players := make(map[int64][]string)
	for _, row := range tp {
		players[row.TeamID] = append(players[row.TeamID], row.Username)
	}

	lite := make([]domain.FixtureLite, 0, len(fixtures))
	for _, f := range fixtures {
		lite = append(lite, domain.FixtureLite{
			Stage:        f.Stage,
			Status:       f.Status,
			HomeTeamID:   int8val(f.HomeTeamID),
			AwayTeamID:   int8val(f.AwayTeamID),
			WinnerTeamID: int8val(f.WinnerTeamID),
		})
	}
	ids := make([]int64, 0, len(teams))
	teamByID := make(map[int64]sqlc.Team, len(teams))
	for _, t := range teams {
		ids = append(ids, t.ID)
		teamByID[t.ID] = t
	}

	sts, err := h.q.ListStandings(ctx)
	if err != nil {
		return nil, err
	}
	standings := make(map[int64]sqlc.Standing, len(sts))
	type gcount struct{ total, done int }
	gc := map[string]*gcount{}
	for _, s := range sts {
		standings[s.TeamID] = s
		if s.GroupLabel.Valid {
			g := s.GroupLabel.String
			if gc[g] == nil {
				gc[g] = &gcount{}
			}
			gc[g].total++
			if s.Played >= 3 {
				gc[g].done++
			}
		}
	}
	groupComplete := make(map[string]bool)
	for g, c := range gc {
		groupComplete[g] = c.total > 0 && c.done == c.total
	}
	// Which third-placed teams advanced is READ from football-data, not computed:
	// a team that appears in a Round-of-32 fixture has qualified.
	bestThird := make(map[int64]bool)
	for _, f := range fixtures {
		if f.Stage == domain.StageR32 {
			if id := int8val(f.HomeTeamID); id != 0 {
				bestThird[id] = true
			}
			if id := int8val(f.AwayTeamID); id != 0 {
				bestThird[id] = true
			}
		}
	}

	return &progressData{
		fixtures:     fixtures,
		teams:        teams,
		players:      players,
		status:        domain.ComputeProgress(lite, ids),
		standings:     standings,
		bestThird:     bestThird,
		groupComplete: groupComplete,
		teamByID:      teamByID,
	}, nil
}

// advStatus returns a group team's advancement verdict. Group rank shows once
// the group is complete; the 3rd-place in/out verdict only shows once the Round
// of 32 is filled by football-data (r32Filled) — before that a 3rd-place team is
// "third" but undecided (thirdDecided=false), not "out".
func advStatus(s sqlc.Standing, ok, complete, r32Filled, bestThird bool) (status string, thirdQual, thirdDecided bool) {
	if !ok || !complete {
		return "pending", false, false
	}
	switch s.Position {
	case 1, 2:
		return "advancing", false, true
	case 3:
		if !r32Filled {
			return "third", false, false
		}
		return "third", bestThird, true
	default:
		return "out", false, true
	}
}

func (h *Handler) bracket(w http.ResponseWriter, r *http.Request) {
	d, err := h.loadProgress(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}

	// Groups (top section): teams grouped by group label, sorted by standing.
	r32Filled := len(d.bestThird) > 0 // R32 has been filled by football-data
	groupMap := make(map[string][]map[string]any)
	groupOrder := []string{}
	for _, t := range d.teams {
		g := "?"
		if t.GroupLabel.Valid {
			g = t.GroupLabel.String
		}
		if _, seen := groupMap[g]; !seen {
			groupOrder = append(groupOrder, g)
		}
		st, hasSt := d.standings[t.ID]
		status, thirdQual, thirdDecided := advStatus(st, hasSt, d.groupComplete[g], r32Filled, d.bestThird[t.ID])
		groupMap[g] = append(groupMap[g], groupTeamView(t, d.players[t.ID], d.status[t.ID], st, hasSt, status, thirdQual, thirdDecided))
	}
	for g := range groupMap {
		rows := groupMap[g]
		sort.SliceStable(rows, func(i, j int) bool {
			pi := posOf(rows[i])
			pj := posOf(rows[j])
			if pi != pj {
				return pi < pj
			}
			return rows[i]["name"].(string) < rows[j]["name"].(string)
		})
	}
	sort.Strings(groupOrder)
	groups := make([]map[string]any, 0, len(groupOrder))
	for _, g := range groupOrder {
		groups = append(groups, map[string]any{"group": g, "teams": groupMap[g]})
	}

	// Knockout bracket: resolved against the fixed official skeleton.
	rounds := h.resolveKnockout(d)

	var champion map[string]any
	for _, t := range d.teams {
		if ts := d.status[t.ID]; ts != nil && ts.Champion {
			champion = teamView(t, d.players[t.ID], ts)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"groups":   groups,
		"rounds":   rounds,
		"champion": champion,
	})
}

func (h *Handler) fixtures(w http.ResponseWriter, r *http.Request) {
	d, err := h.loadProgress(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "db error")
		return
	}
	statusFilter := r.URL.Query().Get("status") // live | finished | upcoming | "" (all)
	stageFilter := r.URL.Query().Get("stage")   // GROUP | R32 | ... | "" (all)
	var teamID int64
	if t := r.URL.Query().Get("team"); t != "" {
		teamID, _ = strconv.ParseInt(t, 10, 64)
	}

	out := make([]map[string]any, 0, len(d.fixtures))
	for _, f := range d.fixtures {
		if !statusMatches(statusFilter, f.Status) {
			continue
		}
		if stageFilter != "" && f.Stage != stageFilter {
			continue
		}
		if teamID != 0 && int8val(f.HomeTeamID) != teamID && int8val(f.AwayTeamID) != teamID {
			continue
		}
		out = append(out, h.fixtureView(f, d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"fixtures": out})
}

func statusMatches(filter, status string) bool {
	switch filter {
	case "", "all":
		return true
	case "finished":
		return domain.IsFinished(status)
	case "live":
		return domain.IsLive(status)
	case "upcoming":
		return !domain.IsFinished(status) && !domain.IsLive(status)
	default:
		return true
	}
}

func teamView(t sqlc.Team, players []string, ts *domain.TeamStatus) map[string]any {
	v := map[string]any{
		"team_id":     t.ID,
		"name":        t.Name,
		"code":        textVal(t.Code),
		"flag_url":    textVal(t.FlagUrl),
		"group_label": textVal(t.GroupLabel),
		"players":     playersOrEmpty(players),
	}
	if ts != nil {
		v["eliminated"] = ts.Eliminated
		v["furthest_stage"] = ts.FurthestStage
		v["champion"] = ts.Champion
	}
	return v
}

func groupTeamView(t sqlc.Team, players []string, ts *domain.TeamStatus, st sqlc.Standing, hasSt bool, status string, thirdQual, thirdDecided bool) map[string]any {
	v := teamView(t, players, ts)
	if hasSt {
		v["position"] = st.Position
		v["played"] = st.Played
		v["won"] = st.Won
		v["draw"] = st.Draw
		v["lost"] = st.Lost
		v["goals_for"] = st.GoalsFor
		v["goals_against"] = st.GoalsAgainst
		v["goal_diff"] = st.GoalDiff
		v["points"] = st.Points
	}
	v["adv_status"] = status
	v["third_qualifying"] = thirdQual
	v["third_decided"] = thirdDecided
	return v
}

func posOf(row map[string]any) int32 {
	if p, ok := row["position"].(int32); ok && p > 0 {
		return p
	}
	return 99
}

func (h *Handler) fixtureView(f sqlc.ListFixturesWithTeamsRow, d *progressData) map[string]any {
	v := fixtureLiteView(f)
	v["home"] = sideView(f.HomeTeamID, f.HomeName, f.HomeCode, f.HomeFlag, d)
	v["away"] = sideView(f.AwayTeamID, f.AwayName, f.AwayCode, f.AwayFlag, d)
	return v
}

// sideView renders one team in a fixture. A nil team id means a not-yet-decided
// (TBD) knockout slot.
func sideView(id pgtype.Int8, name, code, flag pgtype.Text, d *progressData) map[string]any {
	if !id.Valid {
		return map[string]any{"team_id": nil, "name": nil, "tbd": true}
	}
	tid := id.Int64
	v := map[string]any{
		"team_id":  tid,
		"name":     textVal(name),
		"code":     textVal(code),
		"flag_url": textVal(flag),
		"players":  playersOrEmpty(d.players[tid]),
	}
	if ts := d.status[tid]; ts != nil {
		v["eliminated"] = ts.Eliminated
		v["furthest_stage"] = ts.FurthestStage
		v["champion"] = ts.Champion
	}
	return v
}

func fixtureLiteView(f sqlc.ListFixturesWithTeamsRow) map[string]any {
	return map[string]any{
		"id":             f.ID,
		"stage":          f.Stage,
		"round_label":    textVal(f.RoundLabel),
		"group_label":    textVal(f.GroupLabel),
		"status":         f.Status,
		"kickoff_at":     tsVal(f.KickoffAt),
		"home_score":     int4val(f.HomeScore),
		"away_score":     int4val(f.AwayScore),
		"winner_team_id": nullableID(f.WinnerTeamID),
		"home_name":      textVal(f.HomeName),
		"away_name":      textVal(f.AwayName),
	}
}

func nullableID(v pgtype.Int8) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}
