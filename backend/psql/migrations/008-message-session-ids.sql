-- +migrate Up
ALTER TABLE message ADD session_id TEXT DEFAULT '';

-- +migrate Down
ALTER TABLE message DROP IF EXISTS session_id;
