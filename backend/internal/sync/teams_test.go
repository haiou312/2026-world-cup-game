package sync

import "testing"

func TestNormalizeName(t *testing.T) {
	cases := map[string]string{
		"United States":  "usa",
		"Korea Republic": "south korea",
		"Türkiye":        "turkey", // diacritic folded, then aliased
		"Turkiye":        "turkey",
		"Czech Republic": "czechia",
		"Congo DR":       "dr congo",
		"Curaçao":        "curacao", // diacritic folded, no alias needed
		"Curacao":        "curacao",
		"Brazil":         "brazil",
		"  Spain ":       "spain",
	}
	for in, want := range cases {
		if got := normalizeName(in); got != want {
			t.Errorf("normalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFdStageToStage(t *testing.T) {
	cases := map[string]string{
		"GROUP_STAGE":    "GROUP",
		"LAST_32":        "R32",
		"LAST_16":        "R16",
		"QUARTER_FINALS": "QF",
		"SEMI_FINALS":    "SF",
		"THIRD_PLACE":    "THIRD",
		"FINAL":          "FINAL",
	}
	for in, want := range cases {
		if got := fdStageToStage(in); got != want {
			t.Errorf("fdStageToStage(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFdStatus(t *testing.T) {
	cases := []struct{ status, duration, want string }{
		{"SCHEDULED", "REGULAR", "NS"},
		{"TIMED", "REGULAR", "NS"},
		{"IN_PLAY", "REGULAR", "LIVE"},
		{"FINISHED", "REGULAR", "FT"},
		{"FINISHED", "EXTRA_TIME", "AET"},
		{"FINISHED", "PENALTY_SHOOTOUT", "PEN"},
	}
	for _, c := range cases {
		if got := fdStatus(c.status, c.duration); got != c.want {
			t.Errorf("fdStatus(%q,%q) = %q, want %q", c.status, c.duration, got, c.want)
		}
	}
}
