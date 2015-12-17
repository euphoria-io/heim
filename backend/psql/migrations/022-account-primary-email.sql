-- +migrate Up
ALTER TABLE account ADD email TEXT NOT NULL DEFAULT '';

UPDATE account SET email = emails.email
    FROM (SELECT account_id, personal_identity.id AS email FROM personal_identity
        WHERE namespace = 'email' ORDER BY verified) emails
    WHERE account.id = emails.account_id;

-- +migrate Down
ALTER TABLE account DROP IF EXISTS email;
