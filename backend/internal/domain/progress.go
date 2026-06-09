package domain

// FixtureLite is the minimal fixture shape needed to infer team progress.
type FixtureLite struct {
	Stage        string
	Status       string
	HomeTeamID   int64
	AwayTeamID   int64
	WinnerTeamID int64
}

type TeamStatus struct {
	FurthestStage string `json:"furthest_stage"`
	Eliminated    bool   `json:"eliminated"`
	Champion      bool   `json:"champion"`
}

// ComputeProgress derives each team's furthest stage and elimination status
// purely from the fixtures we synced from api-football.
//
// Rules:
//   - A team's furthest stage is the deepest stage it appears in.
//   - In a finished knockout match the loser is eliminated; the FINAL winner is champion.
//   - Once the Round of 32 has real teams, any team still stuck at GROUP is eliminated.
func ComputeProgress(fixtures []FixtureLite, allTeamIDs []int64) map[int64]*TeamStatus {
	status := make(map[int64]*TeamStatus)
	get := func(id int64) *TeamStatus {
		if status[id] == nil {
			status[id] = &TeamStatus{FurthestStage: StageGroup}
		}
		return status[id]
	}
	for _, id := range allTeamIDs {
		get(id)
	}

	knockoutStarted := false

	for _, f := range fixtures {
		if f.Stage == StageR32 && f.HomeTeamID != 0 && f.AwayTeamID != 0 {
			knockoutStarted = true
		}
		for _, tid := range [2]int64{f.HomeTeamID, f.AwayTeamID} {
			if tid == 0 {
				continue
			}
			ts := get(tid)
			if StageRank(f.Stage) > StageRank(ts.FurthestStage) {
				ts.FurthestStage = f.Stage
			}
		}

		// Elimination only from decided knockout matches (not group, not 3rd-place
		// which doesn't eliminate — both teams already lost their semi).
		if f.Stage != StageGroup && f.Stage != StageThird &&
			IsFinished(f.Status) && f.WinnerTeamID != 0 {
			loser := f.HomeTeamID
			if f.WinnerTeamID == f.HomeTeamID {
				loser = f.AwayTeamID
			}
			if loser != 0 {
				get(loser).Eliminated = true
			}
			if f.Stage == StageFinal {
				get(f.WinnerTeamID).Champion = true
			}
		}
	}

	if knockoutStarted {
		for _, ts := range status {
			if ts.FurthestStage == StageGroup {
				ts.Eliminated = true
			}
		}
	}

	return status
}
