-- +goose Up
CREATE TABLE users (
  id            BIGSERIAL PRIMARY KEY,
  username      TEXT UNIQUE NOT NULL,
  password_hash TEXT NOT NULL,
  is_admin      BOOLEAN NOT NULL DEFAULT false,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE teams (
  id          BIGSERIAL PRIMARY KEY,
  api_team_id INTEGER UNIQUE,
  name        TEXT NOT NULL,
  code        TEXT,
  flag_url    TEXT,
  group_label TEXT
);

CREATE TABLE assignments (
  user_id      BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  team_id      BIGINT NOT NULL REFERENCES teams(id),
  round_number INTEGER NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (team_id, round_number)
);

CREATE TABLE fixtures (
  id             BIGSERIAL PRIMARY KEY,
  api_fixture_id INTEGER UNIQUE NOT NULL,
  stage          TEXT NOT NULL,
  round_label    TEXT,
  group_label    TEXT,
  home_team_id   BIGINT REFERENCES teams(id),
  away_team_id   BIGINT REFERENCES teams(id),
  home_score     INTEGER,
  away_score     INTEGER,
  status         TEXT NOT NULL,
  winner_team_id BIGINT REFERENCES teams(id),
  kickoff_at     TIMESTAMPTZ,
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_fixtures_stage ON fixtures(stage);
CREATE INDEX idx_fixtures_kickoff ON fixtures(kickoff_at);

-- +goose Down
DROP TABLE IF EXISTS fixtures;
DROP TABLE IF EXISTS assignments;
DROP TABLE IF EXISTS teams;
DROP TABLE IF EXISTS users;
