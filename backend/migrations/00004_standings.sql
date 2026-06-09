-- +goose Up
CREATE TABLE standings (
  team_id       BIGINT PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
  group_label   TEXT,
  position      INTEGER NOT NULL DEFAULT 0,
  played        INTEGER NOT NULL DEFAULT 0,
  won           INTEGER NOT NULL DEFAULT 0,
  draw          INTEGER NOT NULL DEFAULT 0,
  lost          INTEGER NOT NULL DEFAULT 0,
  goals_for     INTEGER NOT NULL DEFAULT 0,
  goals_against INTEGER NOT NULL DEFAULT 0,
  goal_diff     INTEGER NOT NULL DEFAULT 0,
  points        INTEGER NOT NULL DEFAULT 0,
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS standings;
