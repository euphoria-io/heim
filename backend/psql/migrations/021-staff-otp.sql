-- +migrate Up

CREATE TABLE otp (
    account_id text NOT NULL PRIMARY KEY REFERENCES account(id) ON DELETE CASCADE,
    uri text NOT NULL,
    validated bool DEFAULT false
);

-- +migrate Down

DROP TABLE IF EXISTS otp;
