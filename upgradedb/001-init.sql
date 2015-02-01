-- +migrate Up

CREATE TABLE IF NOT EXISTS room (
    name text,
    founded_by text,
    PRIMARY KEY (name)
);

-- +migrate StatementBegin
-- create index if not exists
DO $$
BEGIN
IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = 'room_founded_by' AND n.nspname = 'public'
) THEN
    CREATE INDEX room_founded_by ON room(founded_by);
END IF;
END$$;
-- +migrate StatementEnd

CREATE TABLE IF NOT EXISTS message (
    room text NOT NULL,
    id text NOT NULL,
    parent text,
    posted timestamp with time zone,
    sender_id text,
    sender_name text,
    content text,
    PRIMARY KEY (room, id)
);

-- +migrate StatementBegin
-- create index if not exists
DO $$
BEGIN
IF NOT EXISTS (
    SELECT 1 FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
        WHERE c.relname = 'message_room_parent' AND n.nspname = 'public'
) THEN
    CREATE INDEX message_room_parent ON message(room, parent);
END IF;
END$$;
-- +migrate StatementEnd
