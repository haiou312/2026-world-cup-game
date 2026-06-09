package domain

import (
	"strings"
	"testing"
)

// TestR32SkeletonStructure locks the hand-typed 2026 Round-of-32 table: any
// accidental edit (duplicate seed, missing group, wrong third-place format)
// fails here instead of silently producing a wrong bracket.
func TestR32SkeletonStructure(t *testing.T) {
	if len(R32Skeleton) != 16 {
		t.Fatalf("R32Skeleton has %d slots, want 16", len(R32Skeleton))
	}

	const groups = "ABCDEFGHIJKL" // the 12 groups
	firsts := map[string]int{}
	seconds := map[string]int{}
	thirds := 0

	check := func(seed string) {
		if IsThirdSeed(seed) {
			thirds++
			letters := strings.TrimSpace(strings.TrimPrefix(seed, "3"))
			if len(letters) != 5 {
				t.Errorf("third seed %q should reference 5 candidate groups, got %d", seed, len(letters))
			}
			seen := map[rune]bool{}
			for _, c := range letters {
				if !strings.ContainsRune(groups, c) {
					t.Errorf("third seed %q has invalid group letter %q", seed, string(c))
				}
				if seen[c] {
					t.Errorf("third seed %q repeats group %q", seed, string(c))
				}
				seen[c] = true
			}
			return
		}
		pos, g := ParseSeed(seed)
		if pos == 0 {
			t.Errorf("seed %q failed to parse", seed)
			return
		}
		if !strings.Contains(groups, g) {
			t.Errorf("seed %q has invalid group %q", seed, g)
		}
		switch pos {
		case 1:
			firsts[g]++
		case 2:
			seconds[g]++
		}
	}

	for _, s := range R32Skeleton {
		check(s.Top)
		check(s.Bottom)
	}

	// Each of the 12 groups contributes exactly one winner and one runner-up.
	if len(firsts) != 12 {
		t.Errorf("got %d distinct group winners, want 12", len(firsts))
	}
	if len(seconds) != 12 {
		t.Errorf("got %d distinct runners-up, want 12", len(seconds))
	}
	for _, gr := range groups {
		g := string(gr)
		if firsts[g] != 1 {
			t.Errorf("group %s winner (1%s) appears %d times, want 1", g, g, firsts[g])
		}
		if seconds[g] != 1 {
			t.Errorf("group %s runner-up (2%s) appears %d times, want 1", g, g, seconds[g])
		}
	}
	if thirds != 8 {
		t.Errorf("got %d third-place slots, want 8", thirds)
	}

	// 12 winners + 12 runners-up + 8 thirds = 32 teams across 16 slots.
	if total := len(firsts) + len(seconds) + thirds; total != 32 {
		t.Errorf("total seeds = %d, want 32", total)
	}
}
