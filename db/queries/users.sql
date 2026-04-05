-- name: CreateUser :one
INSERT INTO users (name, email, password, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 AND is_active = true;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: ListUsers :many
SELECT id, name, email, role, is_active, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: UpdateUserPassword :exec
UPDATE users SET password = $2, updated_at = NOW() WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1;
