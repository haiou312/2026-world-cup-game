-- name: AcquireDrawLock :exec
SELECT pg_advisory_xact_lock(778899);

-- name: CountAssignments :one
SELECT count(*) FROM assignments;

-- name: PickAvailableTeam :one
SELECT id FROM teams
WHERE id NOT IN (SELECT team_id FROM assignments WHERE round_number = $1)
ORDER BY random()
LIMIT 1;

-- name: CreateAssignment :one
INSERT INTO assignments (participant_id, team_id, round_number)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListTeamPlayers :many
SELECT a.team_id, p.name AS username
FROM assignments a
JOIN participants p ON p.id = a.participant_id
ORDER BY a.team_id, p.name;

-- name: GetAssignmentWithTeam :one
SELECT a.participant_id, a.team_id, a.round_number, a.created_at,
       t.name        AS team_name,
       t.code        AS team_code,
       t.flag_url    AS team_flag_url,
       t.group_label AS team_group_label
FROM assignments a
JOIN teams t ON t.id = a.team_id
WHERE a.participant_id = $1;
