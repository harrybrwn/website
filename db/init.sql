CREATE TABLE IF NOT EXISTS "user" (
	id         SERIAL PRIMARY KEY,
	uuid       UUID UNIQUE,
	username   VARCHAR(256),
	email      VARCHAR(256),
	pw_hash    BYTEA,
	totp_code  VARCHAR(32),
	roles      VARCHAR(32)[],
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "request_log" (
	id           SERIAL PRIMARY KEY,
	method       VARCHAR(7), -- longest is OPTIONS
	status       INT,
	ip           INET,
	uri          TEXT,
	referer      TEXT,
	user_agent   TEXT,
	latency      INT,
	error        TEXT,
	requested_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
