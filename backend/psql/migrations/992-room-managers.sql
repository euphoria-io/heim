-- +migrate Up
-- new table and foreign key for room manager

CREATE TABLE room_manager (
    room text NOT NULL,
    account_id text NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    capability_id text NOT NULL,
    FOREIGN KEY (room, capability_id) REFERENCES room_capability(room, capability_id) ON DELETE CASCADE,
    PRIMARY KEY (room, account_id)
);

-- index room_manager by account_id, room
CREATE INDEX room_manager_account_id_room ON room_manager(account_id, room);

-- add nonce to capability table
ALTER TABLE capability ADD nonce bytea;

-- add foreign key so capability deletions cascade
ALTER TABLE room_capability ADD CONSTRAINT capability_fk FOREIGN KEY (capability_id) REFERENCES capability(id) ON DELETE CASCADE;

-- +migrate Down
-- drop new table, column, and foreign key

DROP TABLE room_manager;
ALTER TABLE capability DROP IF EXISTS nonce;
ALTER TABLE room_capability DROP CONSTRAINT IF EXISTS capability_fk;
