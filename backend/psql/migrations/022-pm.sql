-- +migrate Up

CREATE TABLE pm (
    id text NOT NULL PRIMARY KEY,
    initiator text NOT NULL REFERENCES account(id),
    receiver text NOT NULL,
    receiver_mac bytea NOT NULL,
    iv bytea NOT NULL,
    encrypted_system_key bytea NOT NULL,
    encrypted_initiator_key bytea NOT NULL,
    encrypted_receiver_key bytea
);

CREATE INDEX pm_receiver ON pm(receiver);

-- +migrate Down

DROP TABLE IF EXISTS pm;
