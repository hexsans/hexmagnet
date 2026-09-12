-- name: GetTorrent :one
SELECT sqlc.embed(t), s.seeders, s.leechers
FROM torrents t
LEFT JOIN torrent_seeders s ON s.info_hash = t.info_hash
WHERE t.info_hash = $1;

-- name: ListTorrentsByInfoHashes :many
SELECT sqlc.embed(t), s.seeders, s.leechers
FROM torrents t
LEFT JOIN torrent_seeders s ON s.info_hash = t.info_hash
WHERE t.info_hash = ANY($1::text[]);

-- name: ListTorrentsPaginated :many
SELECT sqlc.embed(t), s.seeders, s.leechers
FROM torrents t
LEFT JOIN torrent_seeders s ON s.info_hash = t.info_hash
ORDER BY t.created_at LIMIT $1 OFFSET $2;

-- name: CountTorrents :one
SELECT COUNT(*) FROM torrents;

-- name: CountTorrentsBefore :one
SELECT COUNT(*) FROM torrents WHERE created_at <= $1::timestamptz;

-- name: ListTorrentsPaginatedBefore :many
SELECT sqlc.embed(t), s.seeders, s.leechers
FROM torrents t
LEFT JOIN torrent_seeders s ON s.info_hash = t.info_hash
WHERE t.created_at <= $1::timestamptz ORDER BY t.created_at LIMIT $2 OFFSET $3;

-- name: UpsertTorrent :exec
INSERT INTO torrents (info_hash, name, size, private, files_count, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (info_hash) DO UPDATE SET
  name = EXCLUDED.name,
  size = EXCLUDED.size,
  private = EXCLUDED.private,
  files_count = EXCLUDED.files_count,
  updated_at = NOW();

-- name: UpdateTorrentContent :exec
UPDATE torrents SET
  content_type = $2,
  content_source = $3,
  content_id = $4,
  languages = $5,
  tsv = $6,
  updated_at = NOW()
WHERE info_hash = $1;

-- name: UpdateTorrentSeeders :exec
INSERT INTO torrent_seeders (info_hash, seeders, leechers, updated_at)
VALUES ($1, $2, $3, NOW())
ON CONFLICT (info_hash) DO UPDATE SET
  seeders = GREATEST(torrent_seeders.seeders, EXCLUDED.seeders),
  leechers = GREATEST(torrent_seeders.leechers, EXCLUDED.leechers),
  updated_at = NOW();

-- name: DeleteTorrent :exec
DELETE FROM torrents WHERE info_hash = $1;
