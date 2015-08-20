-- +migrate Up
ALTER TABLE message
    ADD sender_is_staff BOOL DEFAULT false,
    ADD sender_is_manager BOOL DEFAULT false;

-- +migrate Down
ALTER TABLE message
    DROP IF EXISTS sender_is_staff,
    DROP IF EXISTS sender_is_manager;
