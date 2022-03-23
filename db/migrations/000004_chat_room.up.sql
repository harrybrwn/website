CREATE TABLE IF NOT EXISTS chatroom (
	-- ID of the chatroom
	id SERIAL PRIMARY KEY,
	-- Chatroom owner
	owner_id INT,
	-- Name of the chatroom
	name       VARCHAR(255),
	public     BOOLEAN,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chatroom_members (
	-- Room ID
	room      INT,
	user_id   INT,
	-- The id of the last message the user has seen.
	last_seen BIGINT DEFAULT 0,
	UNIQUE (room, user_id)
);

CREATE TABLE IF NOT EXISTS chatroom_messages (
	-- Chat ID
	id         BIGSERIAL PRIMARY KEY,
	-- Room ID
	room       INT,
	user_id    INT,
	body       TEXT,
	-- Messages are created by the client so we should store the client's
	-- timezone.
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS
	ix_chatroom_messages_created_at
	ON chatroom_messages (created_at DESC);
