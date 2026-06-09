package domain

import "testing"

func TestStageRankOrder(t *testing.T) {
	order := []string{StageGroup, StageR32, StageR16, StageQF, StageSF, StageThird, StageFinal}
	for i := 1; i < len(order); i++ {
		if StageRank(order[i]) <= StageRank(order[i-1]) {
			t.Errorf("stage rank not increasing at %s", order[i])
		}
	}
}
