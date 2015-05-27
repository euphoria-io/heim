-- +migrate Up
-- new columns for room key pairs

ALTER TABLE room ADD pk_iv bytea, ADD pk_mac bytea, ADD encrypted_kek bytea, ADD encrypted_private_key bytea, ADD public_key bytea;

-- +migrate Down
-- drop the new columns

ALTER TABLE room DROP IF EXISTS pk_iv, DROP IF EXISTS pk_mac, DROP IF EXISTS encrypted_kek, DROP IF EXISTS encrypted_private_key, DROP IF EXISTS public_key;
