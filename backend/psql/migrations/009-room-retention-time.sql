-- +migrate Up
ALTER TABLE room ADD retention_days INT DEFAULT 0;

-- +migrate Down
ALTER TABLE room DROP IF EXISTS retention_days;