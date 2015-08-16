-- +migrate Up
ALTER TABLE agent ADD blessed BOOL DEFAULT false;
ALTER TABLE room ADD min_agent_age INT DEFAULT 0;

-- +migrate Down
ALTER TABLE agent DROP IF EXISTS blessed;
ALTER TABLE room DROP IF EXISTS min_agent_age;
