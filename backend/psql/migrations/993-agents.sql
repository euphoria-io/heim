-- +migrate Up
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

-- +migrate Down
-- drop new table

DROP TABLE agent;
