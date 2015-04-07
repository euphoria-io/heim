-- +migrate Up
-- Add indexes for message timestamps.

CREATE INDEX message_posted ON message(posted);
CREATE INDEX message_room_posted ON message(room, posted);
CREATE INDEX message_edited ON message(edited);
CREATE INDEX message_room_edited ON message(room, edited);

-- +migrate Down
-- Drop the new indexes.

DROP INDEX message_posted;
DROP INDEX message_room_posted;
DROP INDEX message_edited;
DROP INDEX message_room_edited;
