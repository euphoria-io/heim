-- +migrate Up
-- storage for media objects

-- Table for metadata about a media object (such as an uploaded image or video).
CREATE TABLE media_object (
    id TEXT NOT NULL,
    room TEXT NOT NULL,
    agent_id TEXT NOT NULL,
    storage TEXT NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    updated TIMESTAMP WITH TIME ZONE,
    encryption_key_id TEXT,
    PRIMARY KEY (id)
);

-- Index to fetch all media in a room chronologically.
CREATE INDEX media_object_room_updated ON media_object(room, updated);

-- Index to fetch all media by agent, chronologically.
CREATE INDEX media_object_agent_id_updated ON media_object(agent_id, updated);

-- Table for metadata about a particular transcoding of a media object.
CREATE TABLE media_transcoding (
    media_id TEXT NOT NULL,
    name TEXT NOT NULL,
    uri TEXT NOT NULL,
    content_type TEXT NOT NULL,
    size BIGINT NOT NULL,
    width INT,
    height INT,
    PRIMARY KEY (media_id, name)
);

-- Index to fetch all transcodings by media_id.
CREATE INDEX media_transcoding_media_id ON media_transcoding(media_id);

-- Table for attaching media to messages.
CREATE TABLE message_attachment (
    message_id TEXT NOT NULL,
    media_id TEXT NOT NULL,
    PRIMARY KEY (message_id, media_id)
);

-- +migrate Down
-- drop the new tables

DROP TABLE IF EXISTS media_object;
DROP TABLE IF EXISTS media_transcoding;
DROP TABLE IF EXISTS media_attachment;
