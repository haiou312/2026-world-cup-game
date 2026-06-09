-- name: CreateParticipant :one
INSERT INTO participants (name) VALUES ($1) RETURNING *;

-- name: DeleteParticipant :exec
DELETE FROM participants WHERE id = $1;

-- name: GetParticipant :one
SELECT * FROM participants WHERE id = $1;

-- name: ListParticipants :many
SELECT p.id, p.name,
       a.team_id, a.round_number,
       t.name        AS team_name,
       t.code        AS team_code,
       t.flag_url    AS team_flag_url,
       t.group_label AS team_group_label
FROM participants p
LEFT JOIN assignments a ON a.participant_id = p.id
LEFT JOIN teams t ON t.id = a.team_id
ORDER BY p.name;

-- name: ClearAssignments :exec
DELETE FROM assignments;
