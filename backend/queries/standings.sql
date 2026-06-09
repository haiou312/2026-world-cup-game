-- name: ListStandings :many
SELECT * FROM standings;

-- name: UpsertStanding :exec
INSERT INTO standings (team_id, group_label, position, played, won, draw, lost,
                       goals_for, goals_against, goal_diff, points, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
ON CONFLICT (team_id) DO UPDATE SET
  group_label   = EXCLUDED.group_label,
  position      = EXCLUDED.position,
  played        = EXCLUDED.played,
  won           = EXCLUDED.won,
  draw          = EXCLUDED.draw,
  lost          = EXCLUDED.lost,
  goals_for     = EXCLUDED.goals_for,
  goals_against = EXCLUDED.goals_against,
  goal_diff     = EXCLUDED.goal_diff,
  points        = EXCLUDED.points,
  updated_at    = now();
