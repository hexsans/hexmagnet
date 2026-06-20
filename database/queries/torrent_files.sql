-- name: ListTorrentFiles :many
SELECT * FROM torrent_files WHERE info_hash = $1 ORDER BY "index";

-- name: ListTorrentFilesPaginated :many
SELECT * FROM torrent_files WHERE info_hash = $1 ORDER BY "index" LIMIT $2 OFFSET $3;

-- name: CountTorrentFiles :one
SELECT COUNT(*) FROM torrent_files WHERE info_hash = $1;

-- name: TorrentFileExists :one
SELECT EXISTS(SELECT 1 FROM torrent_files WHERE info_hash = $1 OFFSET $2);

-- name: UpsertTorrentFile :exec
INSERT INTO torrent_files (info_hash, "index", path_parts, size, created_at, updated_at)
VALUES ($1, $2, $3, $4, NOW(), NOW())
ON CONFLICT (info_hash, "index") DO UPDATE SET
  path_parts = EXCLUDED.path_parts,
  size = EXCLUDED.size,
  updated_at = NOW();
