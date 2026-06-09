package sync

import (
	"context"
	"log/slog"
	"strings"
	"unicode"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"worldcup/internal/db/sqlc"
)

type teamCache struct {
	byAPI  map[int32]int64
	byCode map[string]int64
	byName map[string]int64
	group  map[int64]string
	// unmatched collects provider team names that had no seed row this pass, so
	// the sync can surface alias/seed gaps to the admin instead of failing silently.
	unmatched []string
}

// nameAliases maps normalized provider team names to our seeded names so a team
// keeps the same internal id (preserving player assignments) across naming
// differences. Matching by TLA/code is tried first, so this is only a fallback.
// Diacritics are already folded by normalizeName (Curaçao→curacao, Türkiye→turkiye),
// so only genuine spelling/wording differences need an entry here.
var nameAliases = map[string]string{
	"united states":          "usa",
	"korea republic":         "south korea",
	"turkiye":                "turkey",
	"czech republic":         "czechia",
	"congo dr":               "dr congo",
	"dr congo":               "dr congo",
	"bosnia and herzegovina": "bosnia-herzegovina",
	"cape verde islands":     "cape verde",
	"ir iran":                "iran",
}

// foldAccents strips diacritical marks so accented provider names match our
// plain-ASCII seed names without a hardcoded alias for each accent variant.
func foldAccents(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".", "")
	s = foldAccents(s)
	if a, ok := nameAliases[s]; ok {
		return a
	}
	return s
}

// setTeamGroup writes a team's group label (idempotent; skips if unchanged).
func (s *Syncer) setTeamGroup(ctx context.Context, c *teamCache, id int64, grp string) error {
	if id == 0 || c.group[id] == grp {
		return nil
	}
	if err := s.q.SetTeamGroup(ctx, sqlc.SetTeamGroupParams{
		ID:         id,
		GroupLabel: pgtype.Text{String: grp, Valid: true},
	}); err != nil {
		return err
	}
	c.group[id] = grp
	return nil
}

func (s *Syncer) buildTeamCache(ctx context.Context) (*teamCache, error) {
	ts, err := s.q.ListTeams(ctx)
	if err != nil {
		return nil, err
	}
	c := &teamCache{
		byAPI:  make(map[int32]int64),
		byCode: make(map[string]int64),
		byName: make(map[string]int64),
		group:  make(map[int64]string),
	}
	for _, t := range ts {
		c.byName[normalizeName(t.Name)] = t.ID
		if t.Code.Valid && t.Code.String != "" {
			c.byCode[strings.ToUpper(t.Code.String)] = t.ID
		}
		if t.ApiTeamID.Valid {
			c.byAPI[t.ApiTeamID.Int32] = t.ID
		}
		if t.GroupLabel.Valid {
			c.group[t.ID] = t.GroupLabel.String
		}
	}
	return c, nil
}

// resolveTeam maps a football-data team to our internal team id, returning 0 for
// an undecided (TBD) knockout slot. Matched seed rows are enriched with the
// provider id + crest so future syncs match by id directly.
func (s *Syncer) resolveTeam(ctx context.Context, c *teamCache, t fdTeam) (int64, error) {
	if t.ID == 0 && t.Name == "" {
		return 0, nil // TBD knockout slot
	}
	if t.ID != 0 {
		if id, ok := c.byAPI[int32(t.ID)]; ok {
			return id, nil
		}
	}

	crest := pgtype.Text{String: t.Crest, Valid: t.Crest != ""}

	enrich := func(id int64) (int64, error) {
		if err := s.q.SetTeamApiInfo(ctx, sqlc.SetTeamApiInfoParams{
			ID:        id,
			ApiTeamID: pgtype.Int4{Int32: int32(t.ID), Valid: t.ID != 0},
			FlagUrl:   crest,
		}); err != nil {
			return 0, err
		}
		if t.ID != 0 {
			c.byAPI[int32(t.ID)] = id
		}
		return id, nil
	}

	if t.Tla != "" {
		if id, ok := c.byCode[strings.ToUpper(t.Tla)]; ok {
			return enrich(id)
		}
	}
	if id, ok := c.byName[normalizeName(t.Name)]; ok {
		return enrich(id)
	}

	// Unknown team — create a row so match data isn't lost. It won't have
	// group/players; record it so an alias/code can be added by the admin.
	slog.Warn("unmatched team, creating new row", "id", t.ID, "name", t.Name, "tla", t.Tla)
	c.unmatched = append(c.unmatched, t.Name)
	created, err := s.q.InsertTeam(ctx, sqlc.InsertTeamParams{
		Name:       t.Name,
		Code:       pgtype.Text{String: t.Tla, Valid: t.Tla != ""},
		FlagUrl:    crest,
		GroupLabel: pgtype.Text{},
		ApiTeamID:  pgtype.Int4{Int32: int32(t.ID), Valid: t.ID != 0},
	})
	if err != nil {
		return 0, err
	}
	c.byAPI[int32(t.ID)] = created.ID
	c.byCode[strings.ToUpper(t.Tla)] = created.ID
	c.byName[normalizeName(t.Name)] = created.ID
	return created.ID, nil
}
