-- +migrate Up

CREATE TABLE email (
    id text NOT NULL PRIMARY KEY,
    account_id text NOT NULL REFERENCES account(id),
    job_id bigint NOT NULL REFERENCES job_item(id),
    email_type text NOT NULL,
    send_to text NOT NULL,
    send_from text NOT NULL,
    message bytea NOT NULL,
    created timestamp with time zone NOT NULL,
    delivered timestamp with time zone,
    failed timestamp with time zone
);

CREATE INDEX email_account_id_created ON email(account_id, created);
CREATE INDEX email_account_id_email_type_created ON email(account_id, email_type, created);

-- +migrate Down

DROP TABLE IF EXISTS email;
