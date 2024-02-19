CREATE TABLE IF NOT EXISTS former_names (
	id INTEGER PRIMARY KEY,
	name TEXT,
	notification_emails TEXT,
	last_checked DATETIME,
	last_updated_status DATETIME,
	status TEXT
);

CREATE TABLE IF NOT EXISTS users (
	id SERIAL PRIMARY KEY,
	email TEXT NOT NULL UNIQUE,
	hashed_password BLOB NOT NULL
);

CREATE TABLE IF NOT EXISTS user_sessions (
	id VARCHAR(36) PRIMARY KEY,
	user_id TEXT NOT NULL
);