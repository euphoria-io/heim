-- +migrate Up

ALTER TABLE room_capability DROP CONSTRAINT IF EXISTS room_capability_room_account_id_key;

-- +migrate Down
-- N/A
