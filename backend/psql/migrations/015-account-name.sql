-- +migrate Up
ALTER TABLE account ADD name text DEFAULT '';

-- +migrate Down
ALTER TABLE message DROP IF EXISTS name;
