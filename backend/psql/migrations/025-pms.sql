-- +migrate Up

CREATE TABLE nick (
    user_id text NOT NULL,
    room text NOT NULL,
    nick text NOT NULL,
    PRIMARY KEY (user_id, room)
);

CREATE INDEX nick_room_user_id ON nick(room, user_id);

CREATE TABLE pm (
    id text NOT NULL PRIMARY KEY,
    initiator text NOT NULL REFERENCES account(id),
    initiator_nick text NOT NULL,
    receiver text NOT NULL,
    receiver_nick text NOT NULL,
    receiver_mac bytea NOT NULL,
    iv bytea NOT NULL,
    encrypted_system_key bytea NOT NULL,
    encrypted_initiator_key bytea NOT NULL,
    encrypted_receiver_key bytea
);

CREATE INDEX pm_initiator ON pm(initiator);
CREATE INDEX pm_receiver ON pm(receiver);

-- +migrate Down

DROP TABLE IF EXISTS pm;
DROP TABLE IF EXISTS nick;
