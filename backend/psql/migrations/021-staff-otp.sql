-- +migrate Up

CREATE TABLE otp (
    account_id text NOT NULL PRIMARY KEY REFERENCES account(id) ON DELETE CASCADE,
    iv bytea NOT NULL,
    encrypted_key bytea NOT NULL,
    digest bytea NOT NULL,
    encrypted_uri bytea NOT NULL,
    validated bool DEFAULT false
);

-- +migrate Down

DROP TABLE IF EXISTS otp;
