-- name: CreateTenant :one
INSERT INTO tenants (name, webhook_url, api_key)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetTenant :one
SELECT * FROM tenants
WHERE id = $1 LIMIT 1;

-- name: ListTenants :many
SELECT * FROM tenants
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateUser :one
INSERT INTO users (tenant_id, cognito_sub, email, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetUserByCognitoSub :one
SELECT * FROM users
WHERE cognito_sub = $1 LIMIT 1;

-- name: CreateLocation :one
INSERT INTO locations (tenant_id, user_id, latitude, longitude, timestamp)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetLocationsByTenant :many
SELECT * FROM locations
WHERE tenant_id = $1
AND timestamp >= $2
AND timestamp <= $3
ORDER BY timestamp DESC
LIMIT $4 OFFSET $5;
