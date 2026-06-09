-- +goose Up
ALTER TABLE sync_state
  ADD COLUMN last_success_at TIMESTAMPTZ,
  ADD COLUMN last_error      TEXT NOT NULL DEFAULT '',
  ADD COLUMN last_error_at   TIMESTAMPTZ,
  ADD COLUMN last_warnings   TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE sync_state
  DROP COLUMN IF EXISTS last_success_at,
  DROP COLUMN IF EXISTS last_error,
  DROP COLUMN IF EXISTS last_error_at,
  DROP COLUMN IF EXISTS last_warnings;
