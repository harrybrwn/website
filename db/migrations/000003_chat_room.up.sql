CREATE TABLE IF NOT EXISTS chatroom (
	-- ID of the chatroom
	id SERIAL PRIMARY KEY,
	-- Chatroom owner
	owner_id INT,
	-- Name of the chatroom
	name VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS chatroom_members (
	-- Room ID
	room      INT,
	user_id   INT
	-- The id of the last message the user has seen.
	last_seen BIGINT,
);

CREATE TABLE IF NOT EXISTS chatroom_messages (
	-- Chat ID
	id         BIGSERIAL,
	-- Room ID
	room       INT,
	user_id    INT,
	message    TEXT,
	created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);
