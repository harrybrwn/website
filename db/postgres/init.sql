CREATE TABLE IF NOT EXISTS "user" (
	id          SERIAL PRIMARY KEY,
	uuid        UUID UNIQUE,
	username    VARCHAR(256),
	email       VARCHAR(256),
	pw_hash     BYTEA,
	totp_secret VARCHAR(32),
	roles       VARCHAR(32)[],
    created_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
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

CREATE OR REPLACE VIEW logs AS SELECT
	id,
	"method",
	status,
	uri,
	latency/1e6 as latency_ms,
	age(current_timestamp, requested_at),
	ip,
	user_agent,
	referer,
	error
FROM request_log ORDER BY requested_at DESC;

ALTER TABLE request_log
ADD COLUMN IF NOT EXISTS
	user_id UUID;
