-- +migrate Up

CREATE TABLE job_queue (
    name text NOT NULL PRIMARY KEY
);
COMMENT ON TABLE job_queue IS 'Names of queues that have been created.';

CREATE TABLE job_item (
    id bigint NOT NULL PRIMARY KEY,
    queue text NOT NULL REFERENCES job_queue(name),
    job_type text NOT NULL,
    data bytea,
    created timestamp with time zone NOT NULL,
    due timestamp with time zone NOT NULL,
    claimed timestamp with time zone,
    completed timestamp with time zone,
    max_work_duration_seconds integer NOT NULL,
    attempts_made integer DEFAULT 0,
    attempts_remaining integer NOT NULL
);
CREATE INDEX job_item_queue_claimed_completed_attempts_remaining_due_id ON job_item(queue, claimed, completed, attempts_remaining, due, id);
CREATE INDEX job_item_queue_completed ON job_item(queue, completed);
COMMENT ON TABLE job_item IS 'Jobs that have been enqueued (including cancelled or completed), and their current state.';

CREATE TABLE job_log (
    job_id bigint NOT NULL REFERENCES job_item(id),
    attempt integer NOT NULL,
    handler_id text NOT NULL,
    started timestamp with time zone NOT NULL,
    finished timestamp with time zone,
    stolen timestamp with time zone,
    stolen_by text,
    outcome text,
    log bytea,
    PRIMARY KEY (job_id, attempt)
);
COMMENT ON TABLE job_log IS 'Claims of jobs, and their outcome if known.';

-- +migrate StatementBegin
CREATE FUNCTION job_claim(_queue text, _handler_id text) RETURNS SETOF job_item AS
$$
DECLARE
    item job_item%rowtype;
BEGIN
    WITH RECURSIVE jobs AS (
        SELECT (job).*, pg_try_advisory_lock((job).id) AS locked
            FROM (
                SELECT job
                    FROM job_item AS job
                    WHERE job.queue = _queue AND claimed IS NULL AND completed IS NULL AND attempts_remaining > 0
                    ORDER BY due, id
                    LIMIT 1
                ) AS t1
        UNION ALL (
            SELECT (job).*, pg_try_advisory_lock((job).id) AS locked
                FROM (
                    SELECT (
                        SELECT job
                            FROM job_item AS job
                            WHERE job.queue = _queue AND claimed IS NULL AND completed IS NULL AND attempts_remaining > 0
                                AND (due, id) > (job.due, job.id)
                            ORDER BY due, id
                            LIMIT 1
                        ) AS job
                        FROM jobs
                        WHERE jobs.id IS NOT NULL
                        LIMIT 1
                    ) AS t1
                )
            ) SELECT * INTO item FROM jobs WHERE locked LIMIT 1;

    IF item IS NULL THEN
        RETURN;
    END IF;

    item.claimed := NOW();
    UPDATE job_item SET claimed = item.claimed, attempts_made = attempts_made+1, attempts_remaining = attempts_remaining-1 WHERE id = item.id;
    INSERT INTO job_log (job_id, attempt, handler_id, started) VALUES (item.id, item.attempts_made, _handler_id, item.claimed);

    PERFORM pg_advisory_unlock(item.id);
    RETURN NEXT item;
    RETURN;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
COMMENT ON FUNCTION job_claim(text, text) IS 'Claim an unclaimed, uncompleted, uncancelled job and returns its job_item row, or NULL if no jobs are immediately available to claim.';

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION job_steal(_queue text, _handler_id text) RETURNS SETOF job_item AS
$$
DECLARE
    item job_item%rowtype;
BEGIN
    WITH RECURSIVE jobs AS (
        SELECT (job).*, pg_try_advisory_lock((job).id) AS locked
            FROM (
                SELECT job
                    FROM job_item AS job, job_log as jl
                    WHERE job.queue = _queue AND jl.job_id = job.id AND attempt = job.attempts_made-1
                        AND job.completed IS NULL
                        AND jl.started < NOW() - max_work_duration_seconds * interval '1 second'
                        AND jl.handler_id != _handler_id
                    ORDER BY job.due, job.id
                    LIMIT 1
                ) AS t1
        UNION ALL (
            SELECT (job).*, pg_try_advisory_lock((job).id) AS locked
                FROM (
                    SELECT (
                        SELECT job
                            FROM job_item AS job, job_log AS jl
                            WHERE job.queue = _queue AND jl.job_id = job.id AND attempt = job.attempts_made-1
                                AND job.completed IS NULL
                                AND jl.started < NOW() - max_work_duration_seconds * interval '1 second'
                                AND jl.handler_id != _handler_id
                                AND (due, id) > (job.due, job.id)
                            ORDER BY due, id
                            LIMIT 1
                        ) AS job
                    ) AS t1
                )
            ) SELECT * INTO item FROM jobs WHERE locked LIMIT 1;

    IF item IS NULL THEN
        RETURN;
    END IF;

    item.claimed := NOW();
    UPDATE job_item
        SET claimed = item.claimed, attempts_made = attempts_made+1, attempts_remaining = attempts_remaining-1
        WHERE id = item.id;

    UPDATE job_log SET stolen = NOW(), stolen_by = _handler_id WHERE job_id = item.id AND attempt = item.attempts_made-1;
    INSERT INTO job_log (job_id, attempt, handler_id, started) VALUES (item.id, item.attempts_made, _handler_id, item.claimed);

    PERFORM pg_advisory_unlock(item.id);
    RETURN NEXT item;
    RETURN;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
COMMENT ON FUNCTION job_steal(text, text) IS 'Steals an expired outstanding claim from another _handler_id on an uncompleted job. Returns the job_item row of the job, or NULL if there are no claims to steal.';

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION job_complete(_job_id bigint, _attempt integer, _log bytea) RETURNS VOID AS
$$
BEGIN
    PERFORM pg_advisory_lock(_job_id);
    UPDATE job_item SET completed = NOW() WHERE id = _job_id;
    UPDATE job_log SET finished = NOW(), log = _log WHERE job_id = _job_id AND attempt = _attempt;
    PERFORM pg_advisory_unlock(_job_id);
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
COMMENT ON FUNCTION job_complete(bigint, integer, bytea) IS 'Releases a claim on a job and marks it as completed.';

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION job_fail(_job_id bigint, _attempt integer, _error text, _log bytea) RETURNS VOID AS
$$
BEGIN
    PERFORM pg_advisory_lock(_job_id);
    UPDATE job_item SET claimed = NULL WHERE id = _job_id;
    UPDATE job_log SET finished = NOW(), outcome = _error, log = _log WHERE job_id = _job_id AND attempt = _attempt;
    PERFORM pg_advisory_unlock(_job_id);
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
COMMENT ON FUNCTION job_fail(bigint, integer, text, bytea) IS 'Releases a claim on a job but leaves it uncompleted.';

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION job_cancel(_job_id bigint) RETURNS VOID AS
$$
BEGIN
    PERFORM pg_advisory_lock(_job_id);
    UPDATE job_item SET completed = NOW() WHERE id = _job_id;
    PERFORM pg_advisory_unlock(_job_id);
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd
COMMENT ON FUNCTION job_cancel(bigint) IS 'Marks a job is cancelled. Existing claims continue to process and may yet complete the job, but it is no longer available to claim or steal.';


-- +migrate Down
DROP FUNCTION IF EXISTS job_claim(text, text);
DROP FUNCTION IF EXISTS job_steal(text, text);
DROP FUNCTION IF EXISTS job_complete(bigint, integer, bytea);
DROP FUNCTION IF EXISTS job_fail(bigint, integer, text, bytea);
DROP FUNCTION IF EXISTS job_cancel(bigint);
DROP TABLE IF EXISTS job_log;
DROP TABLE IF EXISTS job_item;
DROP TABLE IF EXISTS job_queue;
