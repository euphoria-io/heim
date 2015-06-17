-- +migrate Up
-- new columns for room key pairs

ALTER TABLE room ADD pk_nonce bytea, ADD pk_iv bytea, ADD pk_mac bytea, ADD encrypted_management_key bytea, ADD encrypted_private_key bytea, ADD public_key bytea;

-- +migrate Down
-- drop the new columns

ALTER TABLE room DROP IF EXISTS pk_nonce, DROP IF EXISTS pk_iv, DROP IF EXISTS pk_mac, DROP IF EXISTS encrypted_management_key, DROP IF EXISTS encrypted_private_key, DROP IF EXISTS public_key;
