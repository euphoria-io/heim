-- +migrate Up
-- Add room_manager_capability and update room_capability.

-- add nonce and account_id to capability table
-- TODO: add fk constraint
ALTER TABLE capability
    ADD nonce bytea,
    ADD account_id text DEFAULT '',
    ADD UNIQUE (id, account_id);

-- get capabilities by account_id
CREATE INDEX capability_account_id ON capability(account_id);

-- add account_id and foreign key so capability deletions cascade
ALTER TABLE room_capability
    ADD account_id text,
    ADD UNIQUE (room, account_id),
    ADD CONSTRAINT capability_fk FOREIGN KEY (capability_id) REFERENCES capability(id) ON DELETE CASCADE;

-- add room_manager_capability table
CREATE TABLE room_manager_capability (
    room text NOT NULL REFERENCES room(name) ON DELETE CASCADE,
    capability_id text NOT NULL UNIQUE,
    account_id text NOT NULL,
    granted timestamp with time zone NOT NULL,
    revoked timestamp with time zone,
    PRIMARY KEY (room, capability_id),
    UNIQUE (room, account_id),
    FOREIGN KEY (capability_id, account_id) REFERENCES capability(id, account_id) ON DELETE CASCADE
);

-- get manager capabilities by room ordered by granted, revoked
CREATE INDEX room_manager_capability_room_granted_revoked
    ON room_manager_capability(room, granted, revoked);

-- +migrate Down
-- drop new table, column, and foreign key

DROP TABLE IF EXISTS room_manager_capability;

ALTER TABLE capability
    DROP IF EXISTS nonce,
    DROP IF EXISTS account_id;

ALTER TABLE room_capability DROP CONSTRAINT IF EXISTS capability_fk;
