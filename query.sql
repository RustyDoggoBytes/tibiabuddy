  
-- name: GetFormerNames :many
SELECT 
	name,
	notification_emails, 
	last_checked,
	last_updated_status, 
	status
FROM
	former_names;

-- name: SaveFormerName :exec
INSERT OR REPLACE INTO former_names (
	id, 
	name,
	notification_emails, 
	last_checked, 
	last_updated_status, 
	status
) VALUES (
(SELECT id FROM former_names fn WHERE fn.name = ?), ?, ?, ?, ?, ?);

-- name: DeleteFormerName :exec
DELETE FROM former_names WHERE name = ?;


-- name: CreateUser :one
INSERT INTO users (
	email,
	hashed_password
) VALUES (
	?, ?
)
RETURNING *;

-- name: GetUser :one
SELECT 
	id,
	email,
	hashed_password
FROM 
	users 
WHERE
	email = ?
;

-- name: DeleteUser :exec
DELETE FROM users where id = ?;

-- name: CreateSession :one
INSERT INTO user_sessions (
	id,
	user_id
) VALUES (
	?, ?
)
RETURNING *;

-- name: DeleteSession :exec
DELETE FROM user_sessions where id = ?;

-- name: GetSession :one
SELECT 
	id,
	user_id
FROM
	user_sessions
WHERE
	id = ?
LIMIT 1
;
