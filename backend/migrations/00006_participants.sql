-- +goose Up
-- Switch from password accounts to plain named participants. Teams/fixtures/
-- standings are untouched; only the people model changes.
DROP TABLE IF EXISTS assignments;
-- CASCADE clears FKs that point at users (assignments.user_id, app_settings.updated_by);
-- app_settings itself (and its saved token) is kept.
DROP TABLE IF EXISTS users CASCADE;

CREATE TABLE participants (
  id         BIGSERIAL PRIMARY KEY,
  name       TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE assignments (
  participant_id BIGINT PRIMARY KEY REFERENCES participants(id) ON DELETE CASCADE,
  team_id        BIGINT NOT NULL REFERENCES teams(id),
  round_number   INTEGER NOT NULL,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (team_id, round_number)
);

-- +goose Down
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS participants;

CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  username      TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  is_admin      BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE assignments (
  user_id      BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  team_id      BIGINT NOT NULL REFERENCES teams(id),
  round_number INTEGER NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (team_id, round_number)
);
