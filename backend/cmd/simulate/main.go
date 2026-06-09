// Command simulate fills the database with a deterministic mock of the whole
// tournament (all 104 matches: scores, group standings, knockout bracket,
// champion) and applies it day-by-day — standing in for the real daily api→DB
// sync so you can watch the app evolve. Usage:
//
//	simulate list                 # print the tournament days with an index
//	simulate reset                # back to pre-tournament (all NS, standings 0)
//	simulate through <dayIndex>   # apply cumulative state as of that day
//	simulate full                 # apply the entire tournament (final state)
package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"worldcup/internal/domain"
)

type fixture struct {
	id     int64
	stage  string
	group  string
	homeID int64
	awayID int64
	date   string // YYYY-MM-DD ("" if no kickoff)
	ord    int    // global order by (kickoff_at, id)
}

type result struct {
	homeID, awayID int64
	hg, ag         int
	status         string // FT | PEN
	winnerID       int64  // 0 for a group-stage draw
}

var (
	teamName  = map[int64]string{}
	teamGroup = map[int64]string{}
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: simulate list | reset | through <dayIndex> | full")
		os.Exit(2)
	}
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://wc:change_me_pg@localhost:5432/worldcup?sslmode=disable"
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	must(err)
	defer pool.Close()

	loadTeams(ctx, pool)
	fixtures := loadFixtures(ctx, pool)
	plan := buildPlan(fixtures)
	days := distinctDays(fixtures)

	switch os.Args[1] {
	case "list":
		for i, d := range days {
			n, stages := 0, map[string]int{}
			for _, f := range fixtures {
				if f.date == d {
					n++
					stages[f.stage]++
				}
			}
			fmt.Printf("day %2d  %s  %2d matches  %v\n", i, d, n, stages)
		}
	case "reset":
		reset(ctx, pool)
		fmt.Println("reset to pre-tournament")
	case "full":
		applyThrough(ctx, pool, fixtures, plan, days, len(days)-1)
		fmt.Println("applied full tournament")
	case "through":
		if len(os.Args) < 3 {
			fmt.Println("need day index")
			os.Exit(2)
		}
		n, err := strconv.Atoi(os.Args[2])
		must(err)
		if n < 0 {
			n = 0
		}
		if n >= len(days) {
			n = len(days) - 1
		}
		applyThrough(ctx, pool, fixtures, plan, days, n)
		fmt.Printf("applied through day %d (%s)\n", n, days[n])
	default:
		fmt.Println("unknown command")
		os.Exit(2)
	}
}

// ---------- loading ----------

func loadTeams(ctx context.Context, pool *pgxpool.Pool) {
	rows, err := pool.Query(ctx, `SELECT id, name, COALESCE(group_label,'') FROM teams`)
	must(err)
	defer rows.Close()
	for rows.Next() {
		var id int64
		var name, grp string
		must(rows.Scan(&id, &name, &grp))
		teamName[id] = name
		teamGroup[id] = grp
	}
}

func loadFixtures(ctx context.Context, pool *pgxpool.Pool) []fixture {
	rows, err := pool.Query(ctx, `
		SELECT id, stage, COALESCE(group_label,''),
		       COALESCE(home_team_id,0), COALESCE(away_team_id,0),
		       COALESCE(to_char(kickoff_at,'YYYY-MM-DD'),'')
		FROM fixtures
		ORDER BY kickoff_at NULLS LAST, id`)
	must(err)
	defer rows.Close()
	var fs []fixture
	i := 0
	for rows.Next() {
		var f fixture
		must(rows.Scan(&f.id, &f.stage, &f.group, &f.homeID, &f.awayID, &f.date))
		f.ord = i
		i++
		fs = append(fs, f)
	}
	return fs
}

func distinctDays(fs []fixture) []string {
	seen := map[string]bool{}
	var days []string
	for _, f := range fs {
		if f.date != "" && !seen[f.date] {
			seen[f.date] = true
			days = append(days, f.date)
		}
	}
	sort.Strings(days)
	return days
}

// ---------- simulation ----------

func strength(name string) int {
	powers := map[string]int{
		"Brazil": 38, "Argentina": 37, "France": 37, "Spain": 36, "England": 35,
		"Germany": 33, "Portugal": 33, "Netherlands": 32, "Belgium": 30, "Croatia": 29,
		"Uruguay": 28, "Colombia": 27, "Morocco": 27, "Japan": 25, "USA": 24,
		"Mexico": 24, "Senegal": 24, "Switzerland": 24, "Ecuador": 22, "South Korea": 22,
		"Iran": 21, "Australia": 20, "Ivory Coast": 21, "Egypt": 21,
	}
	s := 40
	if b, ok := powers[name]; ok {
		s += b
	}
	s += int(hsh(name) % 12)
	return s
}

func hsh(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

func goals(diff int, h uint64) int {
	g := int(h % 3) // 0..2
	if diff > 8 {
		g += int((h >> 5) % 2)
	}
	if diff > 20 {
		g++
	}
	if diff < -12 && g > 0 {
		g--
	}
	if g < 0 {
		g = 0
	}
	if g > 5 {
		g = 5
	}
	return g
}

func simMatch(home, away int64, seed int64, knockout bool) result {
	hs, as := strength(teamName[home]), strength(teamName[away])
	hg := goals(hs-as, hsh(fmt.Sprintf("%d-h", seed)))
	ag := goals(as-hs, hsh(fmt.Sprintf("%d-a", seed)))
	r := result{homeID: home, awayID: away, hg: hg, ag: ag, status: "FT"}
	switch {
	case hg > ag:
		r.winnerID = home
	case ag > hg:
		r.winnerID = away
	default:
		if knockout {
			r.status = "PEN"
			if hsh(fmt.Sprintf("%d-p", seed))%2 == 0 {
				r.winnerID = home
			} else {
				r.winnerID = away
			}
		}
	}
	return r
}

func buildPlan(fs []fixture) map[int64]result {
	plan := map[int64]result{}
	byStage := func(stage string) []fixture {
		var out []fixture
		for _, f := range fs {
			if f.stage == stage {
				out = append(out, f)
			}
		}
		return out // already in global order
	}

	for _, f := range byStage("GROUP") {
		plan[f.id] = simMatch(f.homeID, f.awayID, f.id, false)
	}

	st := computeStandings(byStage("GROUP"), plan)
	posMap := map[string]map[int]int64{}
	for id, s := range st {
		if posMap[s.group] == nil {
			posMap[s.group] = map[int]int64{}
		}
		posMap[s.group][s.position] = id
	}
	seedTeam := func(seed string) int64 {
		if len(seed) < 2 {
			return 0
		}
		return posMap[seed[1:]][int(seed[0]-'0')]
	}
	thirds8 := bestThirds(st)

	// Assign R32 per the official skeleton so the data matches the bracket resolver.
	r32 := byStage("R32")
	ti := 0
	for i, f := range r32 {
		if i >= len(domain.R32Skeleton) {
			break
		}
		ss := domain.R32Skeleton[i]
		top := seedTeam(ss.Top)
		var bot int64
		if strings.HasPrefix(ss.Bottom, "3") {
			if ti < len(thirds8) {
				bot = thirds8[ti]
				ti++
			}
		} else {
			bot = seedTeam(ss.Bottom)
		}
		plan[f.id] = simMatch(top, bot, f.id, true)
	}
	propagate(plan, byStage("R32"), byStage("R16"))
	propagate(plan, byStage("R16"), byStage("QF"))
	propagate(plan, byStage("QF"), byStage("SF"))

	sf := byStage("SF")
	if fin := byStage("FINAL"); len(fin) == 1 && len(sf) == 2 {
		plan[fin[0].id] = simMatch(plan[sf[0].id].winnerID, plan[sf[1].id].winnerID, fin[0].id, true)
	}
	if third := byStage("THIRD"); len(third) == 1 && len(sf) == 2 {
		plan[third[0].id] = simMatch(loser(plan[sf[0].id]), loser(plan[sf[1].id]), third[0].id, true)
	}
	return plan
}

func propagate(plan map[int64]result, prev, cur []fixture) {
	for i, f := range cur {
		if 2*i+1 >= len(prev) {
			break
		}
		plan[f.id] = simMatch(plan[prev[2*i].id].winnerID, plan[prev[2*i+1].id].winnerID, f.id, true)
	}
}

func loser(r result) int64 {
	if r.winnerID == r.homeID {
		return r.awayID
	}
	return r.homeID
}

// ---------- standings ----------

type stat struct {
	teamID                                   int64
	group                                    string
	played, won, draw, lost, gf, ga, gd, pts int
	position                                 int
}

func computeStandings(groupFx []fixture, plan map[int64]result) map[int64]*stat {
	m := map[int64]*stat{}
	// Seed every team that belongs to a group so all four always get ranked,
	// even before they have played (otherwise unplayed teams keep stale positions).
	for id, grp := range teamGroup {
		if grp != "" {
			m[id] = &stat{teamID: id, group: grp}
		}
	}
	get := func(id int64) *stat {
		if m[id] == nil {
			m[id] = &stat{teamID: id, group: teamGroup[id]}
		}
		return m[id]
	}
	for _, f := range groupFx {
		r, ok := plan[f.id]
		if !ok {
			continue
		}
		h, a := get(f.homeID), get(f.awayID)
		h.played++
		a.played++
		h.gf += r.hg
		h.ga += r.ag
		a.gf += r.ag
		a.ga += r.hg
		switch {
		case r.hg > r.ag:
			h.won++
			h.pts += 3
			a.lost++
		case r.ag > r.hg:
			a.won++
			a.pts += 3
			h.lost++
		default:
			h.draw++
			a.draw++
			h.pts++
			a.pts++
		}
	}
	for _, s := range m {
		s.gd = s.gf - s.ga
	}
	byGroup := map[string][]*stat{}
	for _, s := range m {
		byGroup[s.group] = append(byGroup[s.group], s)
	}
	for _, list := range byGroup {
		sort.SliceStable(list, func(i, j int) bool { return less(list[i], list[j]) })
		for i, s := range list {
			s.position = i + 1
		}
	}
	return m
}

func less(a, b *stat) bool {
	if a.pts != b.pts {
		return a.pts > b.pts
	}
	if a.gd != b.gd {
		return a.gd > b.gd
	}
	if a.gf != b.gf {
		return a.gf > b.gf
	}
	return teamName[a.teamID] < teamName[b.teamID]
}

// bestThirds returns the best 8 third-placed teams (points → GD → goals scored).
func bestThirds(st map[int64]*stat) []int64 {
	var thirds []*stat
	for _, s := range st {
		if s.position == 3 {
			thirds = append(thirds, s)
		}
	}
	sort.SliceStable(thirds, func(i, j int) bool { return less(thirds[i], thirds[j]) })
	if len(thirds) > 8 {
		thirds = thirds[:8]
	}
	out := make([]int64, 0, len(thirds))
	for _, s := range thirds {
		out = append(out, s.teamID)
	}
	return out
}

// ---------- apply to DB ----------

func reset(ctx context.Context, pool *pgxpool.Pool) {
	_, err := pool.Exec(ctx, `UPDATE fixtures SET home_score=NULL, away_score=NULL, status='NS', winner_team_id=NULL`)
	must(err)
	_, err = pool.Exec(ctx, `UPDATE fixtures SET home_team_id=NULL, away_team_id=NULL WHERE stage <> 'GROUP'`)
	must(err)
	_, err = pool.Exec(ctx, `UPDATE standings SET played=0, won=0, draw=0, lost=0, goals_for=0, goals_against=0, goal_diff=0, points=0`)
	must(err)
}

func applyThrough(ctx context.Context, pool *pgxpool.Pool, fs []fixture, plan map[int64]result, days []string, n int) {
	reset(ctx, pool)
	target := days[n]

	stageFx := map[string][]fixture{}
	for _, f := range fs {
		stageFx[f.stage] = append(stageFx[f.stage], f)
	}
	idx := map[int64]int{}
	for _, list := range stageFx {
		for i, f := range list {
			idx[f.id] = i
		}
	}
	lastGroup := ""
	for _, f := range stageFx["GROUP"] {
		if f.date > lastGroup {
			lastGroup = f.date
		}
	}
	feeders := func(f fixture) []fixture {
		prev := stageFx[map[string]string{"R16": "R32", "QF": "R16", "SF": "QF", "FINAL": "SF", "THIRD": "SF"}[f.stage]]
		if f.stage == "FINAL" || f.stage == "THIRD" {
			if len(prev) >= 2 {
				return []fixture{prev[0], prev[1]}
			}
			return nil
		}
		if i := idx[f.id]; 2*i+1 < len(prev) {
			return []fixture{prev[2*i], prev[2*i+1]}
		}
		return nil
	}
	// A fixture's teams are known once its feeding round is fully played
	// (R32 depends on the entire group stage finishing).
	teamsKnown := func(f fixture) bool {
		switch f.stage {
		case "GROUP":
			return true
		case "R32":
			return lastGroup != "" && lastGroup <= target
		default:
			fe := feeders(f)
			if len(fe) == 0 {
				return false
			}
			for _, x := range fe {
				if x.date == "" || x.date > target {
					return false
				}
			}
			return true
		}
	}

	for _, f := range fs {
		if !teamsKnown(f) {
			continue // stays TBD
		}
		r, ok := plan[f.id]
		if !ok {
			continue
		}
		if f.date != "" && f.date <= target {
			_, err := pool.Exec(ctx, `
				UPDATE fixtures
				SET home_team_id=$2, away_team_id=$3, home_score=$4, away_score=$5,
				    status=$6, winner_team_id=NULLIF($7,0), updated_at=now()
				WHERE id=$1`,
				f.id, r.homeID, r.awayID, r.hg, r.ag, r.status, r.winnerID)
			must(err)
		} else {
			// matchup decided but not yet played: show teams, no score
			_, err := pool.Exec(ctx, `
				UPDATE fixtures
				SET home_team_id=$2, away_team_id=$3, home_score=NULL, away_score=NULL,
				    status='NS', winner_team_id=NULL, updated_at=now()
				WHERE id=$1`,
				f.id, r.homeID, r.awayID)
			must(err)
		}
	}

	var groupUpTo []fixture
	for _, f := range fs {
		if f.stage == "GROUP" && f.date != "" && f.date <= target {
			groupUpTo = append(groupUpTo, f)
		}
	}
	for _, s := range computeStandings(groupUpTo, plan) {
		_, err := pool.Exec(ctx, `
			UPDATE standings
			SET position=$2, played=$3, won=$4, draw=$5, lost=$6,
			    goals_for=$7, goals_against=$8, goal_diff=$9, points=$10, updated_at=now()
			WHERE team_id=$1`,
			s.teamID, s.position, s.played, s.won, s.draw, s.lost, s.gf, s.ga, s.gd, s.pts)
		must(err)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
