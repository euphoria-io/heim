-- +migrate Up

ALTER TABLE stats_sessions_global ADD bot bool NOT NULL DEFAULT FALSE;
ALTER TABLE stats_sessions_per_room ADD bot bool NOT NULL DEFAULT FALSE;

CREATE INDEX stats_sessions_global_bot_sender_id_first_posted_last_posted
    ON stats_sessions_global(bot, sender_id, first_posted, last_posted);

CREATE INDEX stats_sessions_per_room_bot_room_sender_id_first_posted_last_posted
    ON stats_sessions_per_room(bot, room, sender_id, first_posted, last_posted);

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION stats_sessions_global_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_discovered int;
BEGIN
    INSERT INTO stats_sessions_global (
        SELECT m.sender_id, MIN(posted) AS first_posted, MIN(posted)+interval '10 seconds' AS last_posted, m.sender_id LIKE 'bot:%' AS bot
            FROM message m LEFT JOIN stats_sessions_global s
            ON m.sender_id = s.sender_id AND m.posted BETWEEN s.first_posted AND s.last_posted
            WHERE s.first_posted IS NULL AND m.posted >= min_posted AND m.posted < max_posted GROUP BY m.sender_id
    );
    GET DIAGNOSTICS sessions_discovered = ROW_COUNT;
    RETURN sessions_discovered;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION stats_sessions_per_room_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_discovered int;
BEGIN
    INSERT INTO stats_sessions_per_room (
        SELECT m.room, m.sender_id, MIN(posted) AS first_posted, MIN(posted) AS last_posted, m.sender_id LIKE 'bot:%' AS bot
            FROM message m LEFT JOIN stats_sessions_per_room s
            ON m.room = s.room
                AND m.sender_id = s.sender_id
                AND m.posted BETWEEN s.first_posted AND s.last_posted
            WHERE s.first_posted IS NULL AND m.posted >= min_posted AND m.posted < max_posted
            GROUP BY m.room, m.sender_id
    );
    GET DIAGNOSTICS sessions_discovered = ROW_COUNT;
    RETURN sessions_discovered;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down

ALTER TABLE stats_sessions_global DROP IF EXISTS bot;
ALTER TABLE stats_sessions_per_room DROP IF EXISTS bot;

DROP INDEX IF EXISTS stats_sessions_global_bot_sender_id_first_posted_last_posted;
DROP INDEX IF EXISTS stats_sessions_per_room_bot_room_sender_id_first_posted_last_posted;

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION stats_sessions_global_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_discovered int;
BEGIN
    INSERT INTO stats_sessions_global (
        SELECT m.sender_id, MIN(posted) AS first_posted, MIN(posted)+interval '10 seconds' AS last_posted
            FROM message m LEFT JOIN stats_sessions_global s
            ON m.sender_id = s.sender_id AND m.posted BETWEEN s.first_posted AND s.last_posted
            WHERE s.first_posted IS NULL AND m.posted >= min_posted AND m.posted < max_posted GROUP BY m.sender_id
    );
    GET DIAGNOSTICS sessions_discovered = ROW_COUNT;
    RETURN sessions_discovered;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION stats_sessions_per_room_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_discovered int;
BEGIN
    INSERT INTO stats_sessions_per_room (
        SELECT m.room, m.sender_id, MIN(posted) AS first_posted, MIN(posted) AS last_posted
            FROM message m LEFT JOIN stats_sessions_per_room s
            ON m.room = s.room
                AND m.sender_id = s.sender_id
                AND m.posted BETWEEN s.first_posted AND s.last_posted
            WHERE s.first_posted IS NULL AND m.posted >= min_posted AND m.posted < max_posted
            GROUP BY m.room, m.sender_id
    );
    GET DIAGNOSTICS sessions_discovered = ROW_COUNT;
    RETURN sessions_discovered;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
