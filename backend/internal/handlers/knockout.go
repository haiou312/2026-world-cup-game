package handlers

import (
	"worldcup/internal/db/sqlc"
	"worldcup/internal/domain"
)

type koFix struct {
	row    sqlc.ListFixturesWithTeamsRow
	home   int64
	away   int64
	winner int64
}

type koSlot struct {
	topTeam, botTeam int64
	topSeed, botSeed string
	fx               *koFix
}

// resolveKnockout builds the knockout rounds following the fixed 2026 skeleton:
// 1st/2nd seeds come from the (completed) group standings; 3rd-place teams and
// all scores come from the actual synced fixtures, located via the known
// seed as an anchor; winners flow up the fixed tree.
func (h *Handler) resolveKnockout(d *progressData) []map[string]any {
	// group → position → team id (final positions only)
	pos := map[string]map[int]int64{}
	for id, s := range d.standings {
		if !s.GroupLabel.Valid {
			continue
		}
		g := s.GroupLabel.String
		if pos[g] == nil {
			pos[g] = map[int]int64{}
		}
		pos[g][int(s.Position)] = id
	}
	resolveSeed := func(seed string) int64 {
		p, g := domain.ParseSeed(seed)
		if p == 0 || !d.groupComplete[g] {
			return 0
		}
		return pos[g][p]
	}

	ko := map[string][]koFix{}
	for _, f := range d.fixtures {
		switch f.Stage {
		case domain.StageR32, domain.StageR16, domain.StageQF, domain.StageSF, domain.StageThird, domain.StageFinal:
			ko[f.Stage] = append(ko[f.Stage], koFix{
				row:    f,
				home:   int8val(f.HomeTeamID),
				away:   int8val(f.AwayTeamID),
				winner: int8val(f.WinnerTeamID),
			})
		}
	}
	find := func(stage string, team int64) *koFix {
		if team == 0 {
			return nil
		}
		for i := range ko[stage] {
			if ko[stage][i].home == team || ko[stage][i].away == team {
				return &ko[stage][i]
			}
		}
		return nil
	}
	other := func(fx *koFix, t int64) int64 {
		if fx.home == t {
			return fx.away
		}
		return fx.home
	}
	winnerOf := func(s koSlot) int64 {
		if s.fx != nil {
			return s.fx.winner
		}
		return 0
	}
	loserOf := func(s koSlot) int64 {
		if s.fx != nil && s.fx.winner != 0 {
			return other(s.fx, s.fx.winner)
		}
		return 0
	}

	// Round of 32 from the skeleton.
	r32 := make([]koSlot, len(domain.R32Skeleton))
	for i, ss := range domain.R32Skeleton {
		s := koSlot{topSeed: ss.Top, botSeed: ss.Bottom}
		s.topTeam = resolveSeed(ss.Top)
		if domain.IsThirdSeed(ss.Bottom) {
			if s.topTeam != 0 {
				if fx := find(domain.StageR32, s.topTeam); fx != nil {
					s.fx = fx
					s.botTeam = other(fx, s.topTeam)
				}
			}
		} else {
			s.botTeam = resolveSeed(ss.Bottom)
			if s.topTeam != 0 {
				s.fx = find(domain.StageR32, s.topTeam)
			} else if s.botTeam != 0 {
				s.fx = find(domain.StageR32, s.botTeam)
			}
		}
		r32[i] = s
	}

	// resolveByWinners locates the real fixture for a tree slot using a feeding
	// winner as the anchor, then takes BOTH teams straight from football-data so
	// the matchup and score are always real; the skeleton only fixes placement.
	// If the fixture isn't filled yet, it falls back to the known feeding winners.
	resolveByWinners := func(tw, bw int64, stage string) koSlot {
		s := koSlot{topTeam: tw, botTeam: bw}
		anchor := tw
		if anchor == 0 {
			anchor = bw
		}
		if anchor == 0 {
			return s
		}
		fx := find(stage, anchor)
		if fx == nil {
			return s
		}
		s.fx = fx
		switch {
		case tw != 0 && (fx.home == tw || fx.away == tw):
			s.topTeam, s.botTeam = tw, other(fx, tw)
		case bw != 0 && (fx.home == bw || fx.away == bw):
			s.topTeam, s.botTeam = other(fx, bw), bw
		default:
			s.topTeam, s.botTeam = fx.home, fx.away
		}
		return s
	}

	pair := func(prev []koSlot, stage string, n int) []koSlot {
		out := make([]koSlot, n)
		for i := 0; i < n; i++ {
			out[i] = resolveByWinners(winnerOf(prev[2*i]), winnerOf(prev[2*i+1]), stage)
		}
		return out
	}
	r16 := pair(r32, domain.StageR16, 8)
	qf := pair(r16, domain.StageQF, 4)
	sf := pair(qf, domain.StageSF, 2)
	finalSlot := resolveByWinners(winnerOf(sf[0]), winnerOf(sf[1]), domain.StageFinal)
	thirdSlot := resolveByWinners(loserOf(sf[0]), loserOf(sf[1]), domain.StageThird)

	return []map[string]any{
		h.koRound(domain.StageR32, r32, d),
		h.koRound(domain.StageR16, r16, d),
		h.koRound(domain.StageQF, qf, d),
		h.koRound(domain.StageSF, sf, d),
		h.koRound(domain.StageThird, []koSlot{thirdSlot}, d),
		h.koRound(domain.StageFinal, []koSlot{finalSlot}, d),
	}
}

func (h *Handler) koRound(stage string, slots []koSlot, d *progressData) map[string]any {
	fx := make([]map[string]any, 0, len(slots))
	for _, s := range slots {
		fx = append(fx, h.koSlotView(s, stage, d))
	}
	return map[string]any{"stage": stage, "fixtures": fx}
}

func (h *Handler) koSlotView(s koSlot, stage string, d *progressData) map[string]any {
	v := map[string]any{
		"stage":          stage,
		"status":         "NS",
		"home_score":     nil,
		"away_score":     nil,
		"winner_team_id": nil,
		"kickoff_at":     nil,
		"id":             nil,
	}
	if s.fx != nil {
		r := s.fx.row
		v["id"] = r.ID
		v["status"] = r.Status
		v["kickoff_at"] = tsVal(r.KickoffAt)
		v["winner_team_id"] = nullableID(r.WinnerTeamID)
		hs, as := int4val(r.HomeScore), int4val(r.AwayScore)
		if s.fx.home == s.topTeam {
			v["home_score"], v["away_score"] = hs, as
		} else {
			v["home_score"], v["away_score"] = as, hs
		}
	}
	v["home"] = h.koSide(s.topTeam, s.topSeed, d)
	v["away"] = h.koSide(s.botTeam, s.botSeed, d)
	return v
}

func (h *Handler) koSide(teamID int64, seed string, d *progressData) map[string]any {
	if teamID != 0 {
		if t, ok := d.teamByID[teamID]; ok {
			return teamView(t, d.players[teamID], d.status[teamID])
		}
	}
	if seed != "" {
		return map[string]any{"team_id": nil, "seed": seed}
	}
	return map[string]any{"team_id": nil, "tbd": true}
}
