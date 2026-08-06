-- name: GetContent :one
SELECT * FROM content WHERE type = $1 AND source = $2 AND id = $3;

-- name: UpsertContent :exec
INSERT INTO content (type, source, id, title, release_date, adult,
  overview, popularity, vote_average, vote_count, tsv, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
ON CONFLICT (type, source, id) DO UPDATE SET
  title = EXCLUDED.title,
  release_date = EXCLUDED.release_date,
  adult = EXCLUDED.adult,
  overview = EXCLUDED.overview,
  popularity = EXCLUDED.popularity,
  vote_average = EXCLUDED.vote_average,
  vote_count = EXCLUDED.vote_count,
  tsv = EXCLUDED.tsv,
  updated_at = NOW();
