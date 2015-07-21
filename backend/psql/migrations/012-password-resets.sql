-- +migrate Up

-- new table for password reset requests
CREATE TABLE password_reset_request (
    id text NOT NULL PRIMARY KEY,
    account_id text NOT NULL REFERENCES account(id) ON DELETE CASCADE,
    key bytea NOT NULL,
    requested timestamp with time zone NOT NULL,
    expires timestamp with time zone NOT NULL,
    consumed timestamp with time zone,
    invalidated timestamp with time zone
);

-- index on account_id, requested
CREATE INDEX password_reset_request_account_id_requested ON password_reset_request(account_id, requested);

-- +migrate Down

DROP TABLE IF EXISTS password_reset_request;
