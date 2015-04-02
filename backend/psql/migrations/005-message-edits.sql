-- +migrate Up
-- new tables and message columns for supporting edits

CREATE TABLE message_edit_log (
    edit_id TEXT NOT NULL,
    room TEXT NOT NULL,
    message_id TEXT NOT NULL,
    editor_id TEXT,
    previous_edit_id TEXT,
    previous_content TEXT NOT NULL,
    previous_parent TEXT,
    PRIMARY KEY (edit_id)
);

-- Index on room, message_id.
CREATE INDEX message_edit_log_room_message_id_edit_id ON message_edit_log(room, message_id, edit_id);

-- Index on editor_id, room.
CREATE INDEX message_edit_log_editor_id ON message_edit_log(editor_id, room, edit_id);

-- +migrate StatementBegin
ALTER TABLE message
    ADD previous_edit_id TEXT,
    ADD edited TIMESTAMP WITH TIME ZONE,
    ADD deleted TIMESTAMP WITH TIME ZONE;
-- +migrate StatementEnd

-- +migrate Down
-- drop the new tables, drop the message column

DROP TABLE message_edit_log;

-- +migrate StatementBegin
ALTER TABLE message
    DROP IF EXISTS previous_edit_id,
    DROP IF EXISTS edited,
    DROP IF EXISTS deleted;
-- +migrate StatementEnd
