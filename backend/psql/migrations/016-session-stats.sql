-- +migrate Up

CREATE TABLE stats_sessions_analyzed (
    max_posted timestamp with time zone NOT NULL PRIMARY KEY
);

CREATE TABLE stats_sessions_global (
    sender_id text,
    first_posted timestamp with time zone,
    last_posted timestamp with time zone,
    PRIMARY KEY (sender_id, first_posted)
);

CREATE INDEX stats_sessions_global_sender_id_first_posted_last_posted ON stats_sessions_global(sender_id, first_posted, last_posted);

CREATE TABLE stats_sessions_per_room (
    room text,
    sender_id text,
    first_posted timestamp with time zone,
    last_posted timestamp with time zone,
    PRIMARY KEY (room, sender_id, first_posted)
);

CREATE INDEX stats_sessions_per_room_room_sender_id_first_posted_last_posted ON stats_sessions_per_room(room, sender_id, first_posted, last_posted);

-- +migrate StatementBegin
CREATE FUNCTION stats_sessions_analyze() RETURNS timestamp with time zone AS
$$
DECLARE
    prev_max_posted timestamp with time zone;
    next_max_posted timestamp with time zone;
    sessions_touched int;
    total_sessions_touched int;
BEGIN
    -- determine time range to analyze
    SELECT MAX(max_posted) INTO prev_max_posted FROM stats_sessions_analyzed;
    IF prev_max_posted IS NULL THEN
        SELECT MIN(posted)-interval '1 second' INTO prev_max_posted FROM message;
    END IF;
    next_max_posted := DATE_TRUNC('day', prev_max_posted) + interval '2 day';
    IF next_max_posted > NOW() - interval '1 hour' THEN
        next_max_posted := NOW() - interval '1 hour';
    END IF;
    IF next_max_posted <= prev_max_posted THEN
        RETURN NULL;
    END IF;

    RAISE INFO 'Analyzing stats from % to %', prev_max_posted, next_max_posted;

    LOOP
        total_sessions_touched := 0;
        LOOP
            RAISE INFO '  extending global sessions... ';
            sessions_touched := stats_sessions_global_extend(prev_max_posted, next_max_posted);
            RAISE INFO '    extended %', sessions_touched;
            total_sessions_touched := total_sessions_touched + sessions_touched;
            EXIT WHEN sessions_touched = 0;
        END LOOP;
        LOOP
            RAISE INFO '  extending per-room sessions... ';
            sessions_touched := stats_sessions_per_room_extend(prev_max_posted, next_max_posted);
            RAISE INFO '    extended %', sessions_touched;
            total_sessions_touched := total_sessions_touched + sessions_touched;
            EXIT WHEN sessions_touched = 0;
        END LOOP;

        RAISE INFO '  finding sessions... ';
        total_sessions_touched := total_sessions_touched
            + stats_sessions_global_find(prev_max_posted, next_max_posted)
            + stats_sessions_per_room_find(prev_max_posted, next_max_posted);
        RAISE INFO '  total sessions touched: %', total_sessions_touched;
        EXIT WHEN total_sessions_touched = 0;
    END LOOP;

    INSERT INTO stats_sessions_analyzed VALUES (next_max_posted);
    RETURN next_max_posted;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE FUNCTION stats_sessions_global_extend(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_updated int;
BEGIN
    WITH replacements AS (
        SELECT s.sender_id, MIN(s.first_posted) AS first_posted, MAX(m.posted)+interval '10 seconds' AS last_posted
            FROM message m, stats_sessions_global s
            WHERE s.last_posted >= min_posted AND m.sender_id = s.sender_id
                AND m.posted > s.last_posted
                AND m.posted < s.last_posted + interval '5 minutes'
                AND m.posted >= min_posted
                AND m.posted < max_posted
            GROUP BY s.sender_id
        )
    UPDATE stats_sessions_global SET last_posted = replacements.last_posted FROM replacements
        WHERE stats_sessions_global.sender_id = replacements.sender_id AND stats_sessions_global.first_posted = replacements.first_posted;
    GET DIAGNOSTICS sessions_updated = ROW_COUNT;
    RETURN sessions_updated;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE FUNCTION stats_sessions_global_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
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
CREATE FUNCTION stats_sessions_per_room_extend(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
$$
DECLARE
    sessions_updated int;
BEGIN
    WITH replacements AS (
        SELECT s.room, s.sender_id, MIN(s.first_posted) AS first_posted, MAX(m.posted) AS last_posted
            FROM message m, stats_sessions_per_room s
            WHERE s.last_posted >= min_posted AND m.room = s.room AND m.sender_id = s.sender_id
                AND m.posted > s.last_posted
                AND m.posted < s.last_posted + interval '5 minutes'
                AND m.posted >= min_posted
                AND m.posted < max_posted
            GROUP BY s.room, s.sender_id
        )
    UPDATE stats_sessions_per_room SET last_posted = replacements.last_posted FROM replacements
        WHERE stats_sessions_per_room.room = replacements.room
            AND stats_sessions_per_room.sender_id = replacements.sender_id
            AND stats_sessions_per_room.first_posted = replacements.first_posted;
    GET DIAGNOSTICS sessions_updated = ROW_COUNT;
    RETURN sessions_updated;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE FUNCTION stats_sessions_per_room_find(min_posted timestamp with time zone, max_posted timestamp with time zone) RETURNS int AS
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

-- +migrate Down

DROP TABLE IF EXISTS stats_sessions_analyzed;
DROP TABLE IF EXISTS stats_sessions_global;
DROP TABLE IF EXISTS stats_sessions_per_room;
DROP FUNCTION IF EXISTS stats_sessions_analyze();
DROP FUNCTION IF EXISTS stats_sessions_global_find(min_posted timestamp with time zone, max_posted timestamp with time zone);
DROP FUNCTION IF EXISTS stats_sessions_global_extend(min_posted timestamp with time zone, max_posted timestamp with time zone);
DROP FUNCTION IF EXISTS stats_sessions_per_room_find(min_posted timestamp with time zone, max_posted timestamp with time zone);
DROP FUNCTION IF EXISTS stats_sessions_per_room_extend(min_posted timestamp with time zone, max_posted timestamp with time zone);

