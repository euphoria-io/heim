-- +migrate Up
-- storage for keys and capabilities, and apply them to rooms

CREATE TABLE master_key (
    id text NOT NULL,
    encrypted_key bytea NOT NULL,
    iv bytea NOT NULL,
    nonce bytea NOT NULL,
    PRIMARY KEY (id)
);

CREATE TABLE capability (
    id text NOT NULL,
    encrypted_private_data bytea,
    public_data bytea,
    PRIMARY KEY (id)
);

CREATE TABLE room_master_key (
    room text NOT NULL,
    key_id text NOT NULL,
    activated timestamp with time zone NOT NULL,
    expired timestamp with time zone,
    comment text,
    PRIMARY KEY (room, key_id)
);

-- get keys by room ordered by activated, expired
CREATE INDEX room_master_key_room_activated_expired ON room_master_key(room, activated, expired);

CREATE TABLE room_capability (
    room text NOT NULL,
    capability_id text NOT NULL,
    granted timestamp with time zone NOT NULL,
    revoked timestamp with time zone,
    PRIMARY KEY (room, capability_id)
);

-- get capabilities by room ordered by granted, revoked
CREATE INDEX room_capability_room_granted_revoked ON room_capability(room, granted, revoked);

-- designate encryption key and status in message table
ALTER TABLE message ADD encryption_key_id text;

-- +migrate Down
-- drop the new tables, drop the message column

DROP TABLE IF EXISTS master_key;
DROP TABLE IF EXISTS capability;
DROP TABLE IF EXISTS room_master_key;
DROP TABLE IF EXISTS room_capability;
ALTER TABLE message DROP IF EXISTS encryption_key_id;
