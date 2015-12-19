-- +migrate Up

DROP TABLE IF EXISTS virtual_address;
ALTER TABLE message ADD sender_client_address TEXT DEFAULT '';

CREATE TABLE virtual_address (
    room TEXT NOT NULL,
    virtual TEXT NOT NULL,
    real INET NOT NULL,
    created TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (room, virtual),
    UNIQUE (room, real)
);

CREATE SEQUENCE virtual_address_seq MINVALUE -2147483648 MAXVALUE 2147483647 CYCLE OWNED BY virtual_address.room;

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION virtualize_address(_room TEXT, _addr INET) RETURNS TEXT AS
$$
DECLARE
    result TEXT;
BEGIN
    LOOP
        SELECT virtual INTO result FROM virtual_address WHERE room = _room AND real = _addr;
        IF FOUND THEN
            RETURN result;
        END IF;
        IF family(_addr) = 4 THEN
            result = virtualize_inet4(_room, _addr);
        ELSIF family(_addr) = 6 THEN
            result = virtualize_inet6(_room, _addr);
        ELSE
            RAISE EXCEPTION 'no support for virtualizing inet family %', family(_addr);
        END IF;
        BEGIN
            INSERT INTO virtual_address (room, virtual, real, created) VALUES (_room, result, _addr, NOW());
            RETURN result;
        EXCEPTION WHEN unique_violation THEN
            -- We appear to have lost the race, go back to the beginning of the loop and try again.
        END;
    END LOOP;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION virtualize_inet4(_room TEXT, _addr INET) RETURNS TEXT AS
$$
DECLARE
    i TEXT := lpad(to_hex(permute32((nextval('virtual_address_seq') & 2147483647)::INTEGER)), 4, '0');
    n INTEGER := _addr - INET '128.0.0.0';
    u TEXT;
    result TEXT;
BEGIN
    -- If this is a x.x.0.0 address, then just return a virtualized prefix.
    IF n & 65535 = 0 THEN
        RETURN i;
    END IF;

    -- Recursively obtain a stable prefix for the upper 16 bits.
    u := virtualize_address(_room, host(network(set_masklen(_addr, 16)))::INET);

    -- Combine with virtualized suffix.
    RETURN u || ':' || i;
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION virtualize_inet6(_room TEXT, _addr INET) RETURNS TEXT AS
$$
DECLARE
    -- We split an IPv6 address into two parts: the upper 32 bits, and the remainder.
    -- The upper part should be virtualized consistently on a per room basis.
    -- To obtain the upper part, we have to do the equivalent of shifting bits
    -- to the right by 96. We convert this 32-bit value into an IPv4 address and
    -- virtualize it accordingly.
    u_n INTEGER := inet ('::' || trim(trailing ':' from host(network(set_masklen(_addr, 32))))) - inet '::8000:0';
    u TEXT := virtualize_address(_room, inet '128.0.0.0' + u_n);
BEGIN
    RETURN u || ':' || lpad(to_hex(permute32((nextval('virtual_address_seq') & 2147483647)::INTEGER)), 4, '0');
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate StatementBegin
CREATE OR REPLACE FUNCTION permute32(_n INTEGER) RETURNS INTEGER AS
$$
DECLARE
    l1 INTEGER := (_n >> 16) & 65535;
    r1 INTEGER := _n & 65535;
    l2 INTEGER;
    r2 INTEGER;
    i INTEGER := 0;
BEGIN
    WHILE i < 3 LOOP
        l2 := r1;
        r2 := l1 # ((((1366 * r1 + 150889) % 714025) / 714025.0) * 32767)::INTEGER;
        l1 := l2;
        r1 := r2;
        i := i + 1;
    END LOOP;
    RETURN ((r1 << 16) + l1);
END;
$$
LANGUAGE plpgsql;
-- +migrate StatementEnd

-- +migrate Down

ALTER TABLE message DROP IF EXISTS sender_client_address;
DROP TABLE IF EXISTS virtual_address;
DROP FUNCTION IF EXISTS permute32(INTEGER);
DROP FUNCTION IF EXISTS virtualize_address(TEXT, INET);
DROP FUNCTION IF EXISTS virtualize_inet4(TEXT, INET);
