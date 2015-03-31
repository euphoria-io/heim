-- +migrate Up
-- storage for bans

CREATE TABLE banned_agent (
    agent_id text NOT NULL,
    room text, -- null means global ban
    created timestamp with time zone NOT NULL,
    expires timestamp with time zone,
    room_reason text,
    agent_reason text,
    private_reason text,
    UNIQUE (agent_id, room)
);

-- Index to look up agent ban by room.
CREATE INDEX banned_agent_agent_id_room_expires_created ON banned_agent(agent_id, room, expires, created);

-- Index to list rooms an agent is banned in.
CREATE INDEX banned_agent_agent_id_expires_created ON banned_agent(agent_id, expires, created);

-- Index to list banned agents by room.
CREATE INDEX banned_agent_room_expires_created ON banned_agent(room, expires, created);

-- +migrate Down
-- drop the new tables

DROP TABLE IF EXISTS banned_agent;
