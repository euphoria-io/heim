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
    staff bool DEFAULT false
);

CREATE TABLE personal_identity (
    namespace text NOT NULL,
    id text NOT NULL,
    account_id text NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    PRIMARY KEY (namespace, id)
);

-- +migrate Down
-- drop new tables

DROP TABLE personal_identity;
DROP TABLE account;
