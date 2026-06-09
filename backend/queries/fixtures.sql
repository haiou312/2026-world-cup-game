-- name: UpsertFixture :exec
INSERT INTO fixtures (api_fixture_id, stage, round_label, group_label,
                      home_team_id, away_team_id, home_score, away_score,
                      status, winner_team_id, kickoff_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
ON CONFLICT (api_fixture_id) DO UPDATE SET
  stage          = EXCLUDED.stage,
  round_label    = EXCLUDED.round_label,
  group_label    = EXCLUDED.group_label,
  home_team_id   = EXCLUDED.home_team_id,
  away_team_id   = EXCLUDED.away_team_id,
  home_score     = EXCLUDED.home_score,
  away_score     = EXCLUDED.away_score,
  status         = EXCLUDED.status,
  winner_team_id = EXCLUDED.winner_team_id,
  kickoff_at     = EXCLUDED.kickoff_at,
  updated_at     = now();

-- name: ListFixturesWithTeams :many
SELECT f.id, f.api_fixture_id, f.stage, f.round_label, f.group_label,
       f.home_team_id, f.away_team_id, f.home_score, f.away_score,
       f.status, f.winner_team_id, f.kickoff_at,
       ht.name AS home_name, ht.code AS home_code, ht.flag_url AS home_flag,
       at.name AS away_name, at.code AS away_code, at.flag_url AS away_flag
FROM fixtures f
LEFT JOIN teams ht ON ht.id = f.home_team_id
LEFT JOIN teams at ON at.id = f.away_team_id
ORDER BY f.kickoff_at NULLS LAST, f.id;
