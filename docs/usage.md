# Usage

The web UI lives at **http://localhost:3333/** (or wherever `server.port` points). This page walks you through it.

## Dashboard overview

The dashboard is your "is everything working?" screen:

- **Stats panel** — total torrents discovered, content breakdown (movies / TV / music / other), and throughput numbers
- **Crawler overview** — the DHT crawler's activity: how many hashes are being discovered and how fast
- **Queue panel** — pending and in-flight processing jobs (you can watch torrents flow through classification)
- **Activity feed** — recent events

If the numbers are zero but the service is healthy, the crawler may still be building up — see [Troubleshooting — no results](troubleshooting.md#i-see-no-torrents-in-search).

## Searching

The search page is where you find torrents:

- **Free text search** — type a title, e.g. `dune 2021`. Results are ranked by relevance; it tolerates typos and matches partial words
- **Filters** — narrow results by content type (movie / TV / music / …), release year, language, and quality/source where available
- **Sorting** — by relevance, seeders, or newest first
- **Pagination** — browse through large result sets

### What you can search for

HexMagnet indexes everything the crawler discovers, including many torrents that aren't movies or shows — games, apps, eBooks, and more. Use filters to slice it however you like.

## Torrent details

Click any result to open its detail page:

- **Classified metadata** — if classification succeeded, you'll see the title, year, content type, language, episode/season info and release group instead of just a raw filename
- **TMDB enrichment** — when TMDB is configured, the page shows official movie/show metadata
- **File list** — the torrent's files with sizes
- **Actions**:
  - **Download `.torrent`** — downloads the torrent file. If `server.embed_trackers` is configured, extra trackers are embedded to improve your client's connectivity
  - **Copy magnet link**
  - **Reprocess** — re-runs classification on this torrent. Use it when you changed classification settings (LLM, TMDB) and want to re-analyze existing items

## Reprocessing torrents

Reprocess runs a single torrent through the classifier again. It's available per-torrent from the detail page. This is useful when:

- You enabled the LLM classifier or TMDB after a torrent was already indexed
- Classification produced wrong results and you changed filter rules

Reprocessing one torrent is fast. If you change the **search backend** (PostgreSQL → Elasticsearch), all torrents are re-indexed automatically — see [Configuration](configuration.md#storage--where-data-is-kept).

## Config page

The Config page is the recommended way to change settings — no file editing required.

It has four tabs:

| Tab | What you can change |
|---|---|
| **Server** | Listen address and port, log levels, log file settings, trackers to embed in downloads, torrent storage folder |
| **DHT** | DHT port, bootstrap nodes, discovery speed, rescrape interval, request limits, responder behavior |
| **Classifier** | Classification concurrency, torrent filter rules, LLM classification, TMDB enrichment |
| **Storage** | PostgreSQL connection, message queue backend (memory / Kafka), search backend (PostgreSQL / Elasticsearch) and embedding settings |

Everything is explained in [Configuration](configuration.md). Highlights of the save behavior:

- Click **Save** on a tab to apply your changes
- The settings are written to `hexmagnet.yaml` and **take effect immediately** — no restart needed
- The only exceptions are the HTTP listen address (`server.ip` / `server.port`) — those need a restart
- Before saving, HexMagnet **checks your new values**: it tests that folders are writable and that PostgreSQL / Elasticsearch / Kafka / TMDB / LLM are reachable. If a check fails, the save is rejected and the error tells you exactly what to fix (see [Troubleshooting](troubleshooting.md#the-dashboard-rejects-my-config))

## Health check

A quick way to verify the service is alive: open **http://localhost:3333/status** in your browser.

You'll get a small JSON document. Look for the `postgres` check — it should report `up`. If the database is unreachable, the status reflects it.

## If the web UI is missing

The web UI is embedded in the binary. If you built from source with a plain `go build`, make sure you used `make build` (which builds the web UI first). The server logs a clear message when the UI files are missing — see [Troubleshooting](troubleshooting.md#the-web-ui-shows-a-blank-page-or-an-error).
