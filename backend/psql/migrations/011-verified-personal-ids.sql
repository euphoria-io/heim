-- +migrate Up

-- new verified column for personal_identity
ALTER TABLE personal_identity ADD verified boolean DEFAULT false;

-- +migrate Down

ALTER TABLE personal_identity DROP IF EXISTS verified;
