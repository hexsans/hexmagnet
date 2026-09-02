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
| **Torznab** | Servarr/Prowlarr API: enable/disable, API key, path, max results, categories |
| **Webhooks** | Push events to external services: enable/disable, URLs, event types, timeout, retries, base URL, queue size |

Everything is explained in [Configuration](configuration.md). Highlights of the save behavior:

- Click **Save** on a tab to apply your changes
- The settings are written to `hexmagnet.yaml` and **take effect immediately** — no restart needed
- The only exceptions are the HTTP listen address (`server.ip` / `server.port`) — those need a restart
- Before saving, HexMagnet **checks your new values**: it tests that folders are writable and that PostgreSQL / Elasticsearch / Kafka / TMDB / LLM are reachable. If a check fails, the save is rejected and the error tells you exactly what to fix (see [Troubleshooting](troubleshooting.md#the-dashboard-rejects-my-config))

## Health check

A quick way to verify the service is alive: open **http://localhost:3333/status** in your browser.

You'll get a small JSON document. Look for the `postgres` check — it should report `up`. If the database is unreachable, the status reflects it.

## Servarr integration (Lidarr / Radarr / Sonarr / Readarr)

HexMagnet speaks **Torznab**, the standard indexer protocol used by the Servarr
stack. This lets Lidarr, Radarr, Sonarr and Readarr search the crawled index
and auto-download matching content — no plugins or bridges needed.

### 1. Enable the API

On the **Config → Torznab** tab (or in `hexmagnet.yaml`):

- tick **Enabled**
- optionally set an **API key** (recommended if the service is reachable from other hosts)
- optionally restrict **Categories**

Then the endpoint is:

```
http://localhost:3333/torznab/api
```

Verify it works:

```bash
curl "http://localhost:3333/torznab/api?t=capabilities"
# and with an API key:
curl "http://localhost:3333/torznab/api?t=capabilities&apikey=yourkey"
```

You should get an XML `<caps>` document listing the supported search modes
(search, movie-search, tv-search, audio-search, book-search) and categories.

### 2. Connect a Servarr app (e.g. Lidarr)

1. In Lidarr: **Settings → Indexers → Add Indexer → Torznab**
2. URL: `http://hexmagnet:3333/torznab/api` (use the hostname/port you expose)
3. API key: the one you configured (or leave blank)
4. Click **Test** — it should succeed
5. Add an album/artist and press **Search for files** — Lidarr queries the
   indexer, matches releases, and sends the magnet/torrent to your download client

Repeat for Radarr (Movies), Sonarr (TV), and Readarr (Books). For TV shows,
passing `tvdbid`/`tmdbid`/`imdbid` matches content exactly; `season`/`ep`
filtering is done via the torrent name (e.g. `S01E05`).

### 3. (Optional) Prowlarr

If you run **Prowlarr**, add HexMagnet there once and all Servarr apps can sync
it through Prowlarr:

1. Prowlarr: **Settings → Indexers → Add Indexer → Torznab / Newznab**
2. URL: `http://hexmagnet:3333/torznab/api`, API key as above, **Test** it
3. In each Servarr app, use Prowlarr's built-in indexer sync instead of adding HexMagnet directly

> Tip: the more content your crawler has classified, the better the matches.
> Turn on TMDB enrichment (`classifier.tmdb`) for exact movie/TV matching via
> IMDb/TMDB/TVDB IDs.

## Webhook integration

Webhooks let external services react to newly crawled content in real time —
no polling. Every time a torrent is classified and stored, HexMagnet POSTs a
JSON event to each configured URL (see
[Configuration — webhooks](configuration.md#webhooks--pushing-events-to-external-services)).

### 1. Enable and configure

On the **Config → Webhooks** tab (or in `hexmagnet.yaml`):

- tick **Enabled**
- add the endpoint(s) under **URLs** (one per line)
- optionally set **Public Base URL** so the payload includes a `.torrent` download link
- optionally filter what gets delivered: tick **Categories** (content types) and/or add
  **Title Regex** / **Filename Regex** patterns (Go RE2, case-insensitive, one per line)
- optionally add **Headers** (e.g. `Authorization: Bearer <token>`) for endpoints that
  require authentication; header values are masked in the UI and re-saving a masked
  value keeps the stored one
- Save — the change applies immediately

### 2. Example consumer: auto-download all new music to qBittorrent

A tiny script that receives the webhook and adds music torrents to qBittorrent:

```python
#!/usr/bin/env python3
# webhook-consumer.py — run with: python3 webhook-consumer.py
import json
import urllib.request
from http.server import BaseHTTPRequestHandler, HTTPServer

QB_URL = "http://localhost:8080"   # qBittorrent Web UI
QB_USER, QB_PASS = "admin", "adminadmin"

class Hook(BaseHTTPRequestHandler):
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers["Content-Length"])))
        if body.get("content_type") != "music":
            self.send_response(200); self.end_headers(); return

        login = urllib.request.urlopen(f"{QB_URL}/api/v2/auth/login",
            data=f"username={QB_USER}&password={QB_PASS}".encode()).read()
        req = urllib.request.Request(f"{QB_URL}/api/v2/torrents/add",
            data=urllib.parse.urlencode({"urls": body["magnet"]}).encode(),
            headers={"Cookie": f"SID={login.decode()}"})
        urllib.request.urlopen(req)
        print("added:", body["name"])
        self.send_response(200); self.end_headers()

    def log_message(self, *args): pass

HTTPServer(("0.0.0.0", 9000), Hook).serve_forever()
```

Point the webhook URL at `http://<host>:9000`, and every new music torrent
classified by the crawler is sent straight to qBittorrent. Use `content_type`,
`seeders`, `size`, or `languages` in your own filters (e.g. only 4K movies,
only English, only audiobooks...).

> The event's `torrent_url` (when `base_url` is set) downloads the `.torrent`
> file with any configured `embed_trackers`, which can be handed to clients
> that prefer files over magnet links.

## If the web UI is missing

The web UI is embedded in the binary. If you built from source with a plain `go build`, make sure you used `make build` (which builds the web UI first). The server logs a clear message when the UI files are missing — see [Troubleshooting](troubleshooting.md#the-web-ui-shows-a-blank-page-or-an-error).
