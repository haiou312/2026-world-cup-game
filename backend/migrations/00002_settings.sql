-- +goose Up
CREATE TABLE app_settings (
  key        TEXT PRIMARY KEY,
  value      TEXT,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_by BIGINT REFERENCES users(id)
);

-- +goose Down
DROP TABLE IF EXISTS app_settings;
