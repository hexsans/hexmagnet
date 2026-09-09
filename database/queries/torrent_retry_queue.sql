-- name: UpsertTorrentRetryBatch :many
INSERT INTO torrent_retry_queue (info_hash, stage, payload, fail_count, last_error, last_failure_at, next_retry_at, dispatched_at)
SELECT
  unnest(@info_hashes::text[]),
  unnest(@stages::text[]),
  unnest(@payloads::jsonb[]),
  1,
  unnest(@last_errors::text[]),
  NOW(),
  NOW(),
  NULL
ON CONFLICT (info_hash, stage) DO UPDATE SET
  payload = EXCLUDED.payload,
  fail_count = torrent_retry_queue.fail_count + 1,
  last_error = EXCLUDED.last_error,
  last_failure_at = NOW(),
  dispatched_at = NULL
RETURNING info_hash, stage, fail_count;

-- name: BackoffTorrentRetryBatch :exec
UPDATE torrent_retry_queue t
SET next_retry_at = NOW() + make_interval(secs => d.delay_secs)
FROM (
  SELECT unnest(@info_hashes::text[]) AS info_hash,
         unnest(@stages::text[]) AS stage,
         unnest(@delay_secs::int[]) AS delay_secs
) d
WHERE t.info_hash = d.info_hash AND t.stage = d.stage;

-- name: DeleteTorrentRetry :exec
DELETE FROM torrent_retry_queue WHERE info_hash = $1;

-- name: ListDueTorrentRetry :many
SELECT * FROM torrent_retry_queue
WHERE next_retry_at <= NOW()
ORDER BY next_retry_at
LIMIT $1;

-- name: MarkTorrentRetryDispatched :exec
UPDATE torrent_retry_queue
SET next_retry_at = $1,
    dispatched_at = NOW(),
    fail_count = $2
WHERE info_hash = $3 AND stage = $4;

-- name: TouchTorrentRetry :execrows
UPDATE torrent_retry_queue
SET next_retry_at = NOW(),
    dispatched_at = NULL
WHERE info_hash = $1;

-- name: ListTorrentRetryQueue :many
SELECT * FROM torrent_retry_queue
ORDER BY next_retry_at
LIMIT $1 OFFSET $2;

-- name: CountTorrentRetryQueue :one
SELECT COUNT(*) FROM torrent_retry_queue;

-- name: ClearTorrentRetryQueue :exec
DELETE FROM torrent_retry_queue;
