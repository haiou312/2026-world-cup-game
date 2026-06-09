-- name: CountTeams :one
SELECT count(*) FROM teams;

-- name: ListTeams :many
SELECT * FROM teams ORDER BY group_label, name;

-- name: InsertTeam :one
INSERT INTO teams (name, code, flag_url, group_label, api_team_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SetTeamApiInfo :exec
UPDATE teams
SET api_team_id = $2,
    flag_url    = COALESCE($3, flag_url)
WHERE id = $1;

-- name: SetTeamGroup :exec
UPDATE teams SET group_label = $2 WHERE id = $1;
