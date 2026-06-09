-- name: GetSetting :one
SELECT value FROM app_settings WHERE key = $1;

-- name: UpsertSetting :exec
INSERT INTO app_settings (key, value, updated_by, updated_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (key) DO UPDATE
  SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now();
