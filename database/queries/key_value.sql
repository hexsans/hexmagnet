-- name: GetKeyValue :one
SELECT * FROM key_value WHERE key = $1;

-- name: UpsertKeyValue :exec
INSERT INTO key_value (key, value, created_at, updated_at)
VALUES ($1, $2, NOW(), NOW())
ON CONFLICT (key) DO UPDATE SET
  value = EXCLUDED.value,
  updated_at = NOW();
