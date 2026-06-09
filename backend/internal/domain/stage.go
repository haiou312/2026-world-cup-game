package domain

const (
	StageGroup = "GROUP"
	StageR32   = "R32"
	StageR16   = "R16"
	StageQF    = "QF"
	StageSF    = "SF"
	StageThird = "THIRD"
	StageFinal = "FINAL"
)

var stageOrder = map[string]int{
	StageGroup: 0,
	StageR32:   1,
	StageR16:   2,
	StageQF:    3,
	StageSF:    4,
	StageThird: 5,
	StageFinal: 6,
}

// KnockoutStages is the bracket order (group stage shown separately).
var KnockoutStages = []string{StageR32, StageR16, StageQF, StageSF, StageThird, StageFinal}

func StageRank(s string) int { return stageOrder[s] }

// IsFinished reports whether a match status (our internal code) means the match is over.
func IsFinished(status string) bool {
	switch status {
	case "FT", "AET", "PEN":
		return true
	default:
		return false
	}
}

func IsLive(status string) bool {
	switch status {
	case "1H", "HT", "2H", "ET", "BT", "P", "LIVE", "INT", "SUSP":
		return true
	default:
		return false
	}
}
