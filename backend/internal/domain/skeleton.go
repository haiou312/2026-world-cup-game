package domain

import "strings"

// SeedSlot is one Round-of-32 matchup in seed terms (e.g. {"1E", "3 ABCDF"}).
type SeedSlot struct {
	Top    string
	Bottom string
}

// R32Skeleton is the fixed 2026 Round-of-32 bracket in official poster order:
// the left half top→bottom (8), then the right half top→bottom (8). Deeper
// rounds pair adjacent slots — R16[i] = winners of R32[2i] and R32[2i+1], etc.
var R32Skeleton = []SeedSlot{
	{"1E", "3 ABCDF"}, {"1I", "3 CDFGH"}, {"2A", "2B"}, {"1F", "2C"},
	{"2K", "2L"}, {"1H", "2J"}, {"1D", "3 BEFIJ"}, {"1G", "3 AEHIJ"},
	{"1C", "2F"}, {"2E", "2I"}, {"1A", "3 CEFHI"}, {"1L", "3 EHIJK"},
	{"1J", "2H"}, {"2D", "2G"}, {"1B", "3 EFGIJ"}, {"1K", "3 DEIJL"},
}

// IsThirdSeed reports whether a seed code refers to a third-placed team.
func IsThirdSeed(seed string) bool { return strings.HasPrefix(seed, "3") }

// ParseSeed returns (position, group) for a "1E"/"2A" seed, or (0, "") for a
// third-place seed (which can't be resolved from a single group).
func ParseSeed(seed string) (int, string) {
	if len(seed) < 2 || seed[0] < '1' || seed[0] > '2' {
		return 0, ""
	}
	return int(seed[0] - '0'), seed[1:]
}
