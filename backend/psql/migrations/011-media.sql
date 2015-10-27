-- +migrate Up

-- new tables for media

CREATE TABLE media (
	media_id TEXT NOT NULL PRIMARY KEY,
	room TEXT NOT NULL,
	storage TEXT NOT NULL,
	uploader TEXT NOT NULL,
	created TIMESTAMP with time zone NOT NULL,
	uploaded TIMESTAMP with time zone
);

CREATE TABLE transcoding (
	media_id TEXT,
	parent_id TEXT NOT NULL PRIMARY KEY,
	name TEXT,
	uri text,
	content_type TEXT,
	size INTEGER,
	width INTEGER,
	height INTEGER
);

CREATE INDEX room_created ON media(room, created);
CREATE INDEX uploader_created ON media(uploader, created);
CREATE INDEX room_uploader_created ON media(room, uploader, created);