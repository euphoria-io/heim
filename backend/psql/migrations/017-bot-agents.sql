-- +migrate Up

ALTER TABLE agent ADD bot bool NOT NULL DEFAULT false;

-- +migrate Down

ALTER TABLE agent DROP IF EXISTS bot;
