package domain

import "testing"

func TestComputeProgress(t *testing.T) {
	fx := []FixtureLite{
		{Stage: StageR32, Status: "FT", HomeTeamID: 1, AwayTeamID: 2, WinnerTeamID: 1},
		{Stage: StageFinal, Status: "FT", HomeTeamID: 1, AwayTeamID: 3, WinnerTeamID: 1},
	}
	st := ComputeProgress(fx, []int64{1, 2, 3, 4})

	if st[1].Eliminated || !st[1].Champion {
		t.Errorf("team1 should be champion, alive: %+v", st[1])
	}
	if st[1].FurthestStage != StageFinal {
		t.Errorf("team1 furthest = %s, want FINAL", st[1].FurthestStage)
	}
	if !st[2].Eliminated {
		t.Error("team2 lost R32, should be eliminated")
	}
	if !st[3].Eliminated {
		t.Error("team3 lost final, should be eliminated")
	}
	if !st[4].Eliminated {
		t.Error("team4 never reached knockout, should be eliminated once R32 started")
	}
}

func TestComputeProgressGroupStageAllAlive(t *testing.T) {
	// Only unfinished group matches → nobody eliminated yet.
	fx := []FixtureLite{
		{Stage: StageGroup, Status: "NS", HomeTeamID: 1, AwayTeamID: 2},
	}
	st := ComputeProgress(fx, []int64{1, 2})
	if st[1].Eliminated || st[2].Eliminated {
		t.Error("no team should be eliminated during group stage")
	}
}
