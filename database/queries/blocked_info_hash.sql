-- name: GetBlockedInfoHashes :many
SELECT info_hash FROM blocked_info_hash WHERE info_hash = ANY($1::text[]);

-- name: GetBlockedInfoHash :one
SELECT * FROM blocked_info_hash WHERE info_hash = $1;

-- name: UpsertBlockedInfoHash :exec
INSERT INTO blocked_info_hash (info_hash, reason, blocked_at)
VALUES ($1, $2, NOW())
ON CONFLICT (info_hash) DO UPDATE SET
  reason = EXCLUDED.reason;

-- name: DeleteBlockedInfoHash :exec
DELETE FROM blocked_info_hash WHERE info_hash = $1;

-- name: ListBlockedInfoHashes :many
SELECT * FROM blocked_info_hash ORDER BY blocked_at DESC;

-- name: CountBlockedInfoHashes :one
SELECT COUNT(*) FROM blocked_info_hash;
