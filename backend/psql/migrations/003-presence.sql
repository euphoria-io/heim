-- +migrate Up
-- storage for presence

CREATE TABLE presence (
    room text NOT NULL,
    topic text NOT NULL,
    server_id text NOT NULL,
    server_era text NOT NULL,
    session_id text NOT NULL,
    updated timestamp with time zone NOT NULL,
    key_id text,
    fact bytea,
    PRIMARY KEY (room, topic, server_id, server_era, session_id)
);

-- Index to help clean up of old facts.
CREATE INDEX presence_updated ON presence(updated);

-- Index to help servers clean up after restarts.
CREATE INDEX presence_server_id_server_era_updated ON presence(server_id, server_era, updated);

-- Index to get full presence state of a room by topic.
CREATE INDEX presence_room_topic_updated ON presence(room, topic, updated);

-- Index to get full presence state of a session.
CREATE INDEX presence_session_id ON presence(session_id, updated);

-- Add server attributes to message sender.
ALTER TABLE message ADD server_id text DEFAULT '', ADD server_era text DEFAULT '';

-- +migrate Down
-- drop the new tables

DROP TABLE IF EXISTS presence;
ALTER TABLE message DROP IF EXISTS server_id, DROP IF EXISTS server_era;
