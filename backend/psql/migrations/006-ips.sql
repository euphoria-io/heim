-- +migrate Up
-- new tables for working with client IPs

CREATE TABLE session_log (
    session_id TEXT NOT NULL,
    ip TEXT NOT NULL,
    room TEXT NOT NULL,
    user_agent TEXT,
    connected TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (session_id)
);

-- Index to look up sessions by IP.
CREATE INDEX session_log_ip_connected ON session_log(ip, connected);

-- Index to look up sessions by room.
CREATE INDEX session_log_room_connected ON session_log(room, connected);

-- Index to look up sessions by IP within room.
CREATE INDEX session_log_room_ip_connected ON session_log(room, ip, connected);

CREATE TABLE banned_ip (
    ip TEXT NOT NULL,
    room TEXT, -- null means global ban
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    expires TIMESTAMP WITH TIME ZONE,
    reason TEXT,
    UNIQUE (ip, room)
);

-- Index to look up IP ban by room.
CREATE INDEX banned_ip_ip_room_expires_created ON banned_ip(ip, room, expires, created);

-- Index to list rooms an IP is banned in.
CREATE INDEX banned_ip_ip_expires_created ON banned_ip(ip, expires, created);

-- Index to list banned IPs by room.
CREATE INDEX banned_ip_room_expires_created ON banned_ip(room, expires, created);

-- +migrate Down
-- drop the new tables

DROP TABLE session_log;
DROP TABLE banned_ip;
