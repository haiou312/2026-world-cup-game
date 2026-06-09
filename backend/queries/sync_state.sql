-- name: GetSyncState :one
SELECT * FROM sync_state WHERE id = 1;

-- name: UpdateSyncState :exec
UPDATE sync_state
SET last_synced_at = $1,
    today_done      = $2,
    api_calls_today = $3,
    calls_date      = $4
WHERE id = 1;

-- name: RecordSyncSuccess :exec
UPDATE sync_state
SET last_success_at = $1,
    last_error      = '',
    last_warnings   = $2
WHERE id = 1;

-- name: RecordSyncError :exec
UPDATE sync_state
SET last_error    = $1,
    last_error_at = $2
WHERE id = 1;
