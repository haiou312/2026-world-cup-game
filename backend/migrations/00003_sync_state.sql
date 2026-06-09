-- +goose Up
CREATE TABLE sync_state (
  id              INTEGER PRIMARY KEY DEFAULT 1,
  last_synced_at  TIMESTAMPTZ,
  today_done      BOOLEAN NOT NULL DEFAULT false,
  api_calls_today INTEGER NOT NULL DEFAULT 0,
  calls_date      DATE,
  CONSTRAINT sync_state_singleton CHECK (id = 1)
);

INSERT INTO sync_state (id) VALUES (1) ON CONFLICT (id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS sync_state;
