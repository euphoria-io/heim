-- +migrate Up
ALTER TABLE account ADD email TEXT NOT NULL DEFAULT '';

UPDATE account SET email = (
    SELECT personal_identity.id FROM personal_identity
        WHERE account_id = account.id AND namespace = 'email' and verified IS TRUE);

-- +migrate Down
ALTER TABLE account DROP IF EXISTS email;
