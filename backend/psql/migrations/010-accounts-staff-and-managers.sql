-- +migrate Up

-- new tables for accounts

CREATE TABLE account (
    id text NOT NULL PRIMARY KEY,
    nonce bytea NOT NULL,
    mac bytea NOT NULL,
    encrypted_system_key bytea NOT NULL,
    encrypted_user_key bytea NOT NULL,
    encrypted_private_key bytea NOT NULL,
    public_key bytea NOT NULL,
    staff_capability_id text REFERENCES capability(id) ON DELETE SET NULL
);

CREATE TABLE personal_identity (
    namespace text NOT NULL,
    id text NOT NULL,
    account_id text NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    PRIMARY KEY (namespace, id)
);

-- new room columns for key pair and grants
ALTER TABLE room ADD pk_nonce bytea, ADD pk_iv bytea, ADD pk_mac bytea, ADD encrypted_management_key bytea, ADD encrypted_private_key bytea, ADD public_key bytea;

-- add nonce and account_id to capability table
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

-- new table for agent cookie tracking
CREATE TABLE agent (
    id text NOT NULL PRIMARY KEY,
    iv bytea NOT NULL,
    mac bytea NOT NULL,
    encrypted_client_key bytea NOT NULL,
    account_id text REFERENCES account(id) ON DELETE CASCADE,
    created timestamp with time zone NOT NULL
);

-- index agent by account_id
CREATE INDEX agent_account_id ON agent(account_id);

-- add foreign key reference from room_master_key onto master_key
ALTER TABLE room_master_key
    ADD CONSTRAINT master_key_fk FOREIGN KEY (key_id) REFERENCES master_key(id) ON DELETE CASCADE;

-- +migrate Down
-- Undo everything!

DROP TABLE IF EXISTS agent;
DROP TABLE IF EXISTS personal_identity;
DROP TABLE IF EXISTS account;
DROP TABLE IF EXISTS room_manager_capability;

ALTER TABLE capability
    DROP IF EXISTS nonce,
    DROP IF EXISTS account_id;

ALTER TABLE room
    DROP IF EXISTS pk_nonce,
    DROP IF EXISTS pk_iv,
    DROP IF EXISTS pk_mac,
    DROP IF EXISTS encrypted_management_key,
    DROP IF EXISTS encrypted_private_key,
    DROP IF EXISTS public_key;

ALTER TABLE room_capability
    DROP IF EXISTS account_id,
    DROP CONSTRAINT IF EXISTS capability_fk;

ALTER TABLE room_master_key
    DROP CONSTRAINT IF EXISTS master_key_fk;
