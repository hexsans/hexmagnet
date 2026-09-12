# Configuration

This page explains every setting HexMagnet has, in plain language. You'll find:

1. [Where the config file lives](#where-the-config-file-lives)
2. [Three ways to configure](#three-ways-to-configure)
3. [How settings are combined](#how-settings-are-combined)
4. [All settings, group by group](#settings-reference)

---

## Where the config file lives

HexMagnet uses a YAML file as its runtime config. The default path is:

```
./hexmagnet.yaml
```

(right next to the binary, or in the container's `/app` folder — see [Installation](installation.md)).

- **On first start the file is created automatically** with all defaults and a header comment — you don't have to write anything by hand
- The dashboard's Config page reads and writes exactly this file
- To use a different location, set the environment variable `HEXMAGNET_CONFIG_FILE` or pass `-c /path/to/config.yaml` when starting the binary

> The parent folder of the config file must exist before startup — HexMagnet won't create it.

## Three ways to configure

### 1. Dashboard (recommended)

Open the web UI → **Config** page. Every setting below can be changed there, and the dashboard saves to `hexmagnet.yaml` for you. It also validates your input before saving — see [Troubleshooting](troubleshooting.md#the-dashboard-rejects-my-config).

### 2. Config file

Edit `hexmagnet.yaml` (or your custom path) by hand. Settings in the file are validated at startup; if something is invalid, the service prints the exact problem and exits.

### 3. Environment variables

Every config key can also be set via an environment variable. The rule is simple:

> Take the dotted path, uppercase it, replace dots with underscores: `server.port` → `SERVER_PORT`.

Examples:

| Config key | Env var |
|---|---|
| `server.port` | `SERVER_PORT` |
| `dht.bootstrap_nodes` | `DHT_BOOTSTRAP_NODES` |
| `dht.requester.request_limit` | `DHT_REQUESTER_REQUEST_LIMIT` |
| `classifier.tmdb.access_token` | `CLASSIFIER_TMDB_ACCESS_TOKEN` |
| `storage.postgres.host` | `STORAGE_POSTGRES_HOST` |
| `storage.queue.backend` | `STORAGE_QUEUE_BACKEND` |
| `storage.search.elasticsearch.embedding.endpoint` | `STORAGE_SEARCH_ELASTICSEARCH_EMBEDDING_ENDPOINT` |

Formatting rules:

- Lists are comma-separated: `DHT_BOOTSTRAP_NODES="router.utorrent.com:6881,router.bittorrent.com:6881"`
- Booleans: `true` / `false` (or `1` / `0`)
- **Durations must use Go duration format** — e.g. `60s`, `5m`, `1h30m`. A bare number like `60` will be rejected with an error explaining this
- Unknown variable names are simply ignored — no error

## How settings are combined

Values are merged in this order (later sources win):

1. **Compiled-in defaults**
2. **`hexmagnet.yaml`** (or `$HEXMAGNET_CONFIG_FILE`)
3. **`./config.yml`** — an optional local override file next to the binary (fine to ignore; exists for advanced setups)
4. **Environment variables**

So an env var always beats the config file, and the config file always beats defaults. You only need to write the settings you want to change.

Unknown keys in YAML are silently ignored, so an old config file stays compatible with newer versions.

---

## Settings reference

### `server` — HTTP server and logging

| Key | Default | What it does |
|---|---|---|
| `server.ip` | `""` | Listen address. `""` = all interfaces (IPv4 + IPv6). Use `0.0.0.0` for IPv4 only. **Changing this requires a restart.** |
| `server.port` | `3333` | Web UI / API port. **Changing this requires a restart.** |
| `server.log.console_level` | `info` | How much the console (stdout) logs: `debug`, `info`, `warn`, `error`. `debug` is verbose. |
| `server.log.file_output_level` | `off` | Also write logs to a file: `debug`, `info`, `warn`, `error`, or `off` to disable file logging. |
| `server.log.file_rotator.path` | `./logs` | Folder where log files are written. Required when file logging is enabled — the folder must exist and be writable, or saves will be rejected. |
| `server.log.file_rotator.max_backups` | `5` | How many rotated log files to keep. `0` = unlimited. |
| `server.log.file_rotator.format` | `text` | Log file format: `text` or `json`. |
| `server.embed_trackers` | `[]` | Extra tracker announce URLs to embed into downloaded `.torrent` files (list of URLs). Helps your download client find peers. |
| `server.torrent_file_path` | `./data/torrents` | Folder where `.torrent` files are stored on disk. Must be writable. |

### `dht` — BitTorrent network (crawling)

| Key | Default | What it does |
|---|---|---|
| `dht.port` | `3334` | UDP port for the DHT node (and TCP for metadata downloads). Must be reachable from the internet for crawling to work well. |
| `dht.bootstrap_nodes` | 6 well-known nodes | Initial DHT nodes used to join the network (list of `host:port`). The defaults work for most users. |
| `dht.reseed_bootstrap_nodes_interval` | `1m0s` | How often to re-query bootstrap nodes to stay connected. **Duration format** — e.g. `60s`. |
| `dht.requester.request_limit` | `50` | Maximum outgoing requests per second (metadata + DHT). `0` = unlimited. Lower if your network or machine struggles. |
| `dht.requester.rescrape_threshold` | `2592000` | How long before a known torrent is re-checked for new peers (seconds). Default = 30 days. |
| `dht.requester.hash_discover_limit` | `10` | Max new hashes discovered per scrape cycle. Higher = faster discovery, more load. |
| `dht.responder.enabled` | `true` | Respond to DHT queries from other nodes. Disable if you don't want to help the network. |
| `dht.responder.global_rate_limit` | `50` | Max incoming DHT queries per second total. `0` = unlimited. |
| `dht.responder.per_ip_rate_limit` | `1` | Max incoming DHT queries per second from a single IP. `0` = unlimited. |

### `classifier` — turning filenames into content

| Key | Default | What it does |
|---|---|---|
| `classifier.concurrency` | `10` | How many torrents can be classified at once. Lower it on weak machines. |
| `classifier.torrent_filter.mode` | `off` | Pre-classification filter: `off` = classify everything; `process` = only process torrents matching the patterns below; `discard` = throw away matching torrents. |
| `classifier.torrent_filter.title_patterns` | `[]` | Regex patterns matched against the torrent name (used with `process` / `discard`). |
| `classifier.torrent_filter.filename_patterns` | `[]` | Regex patterns matched against file names (checked when the title doesn't match). |

### `classifier.tmdb` — movie / TV metadata enrichment

| Key | Default | What it does |
|---|---|---|
| `classifier.tmdb.enabled` | `true` | Enable TMDB enrichment. Note: without a token below, nothing will happen. |
| `classifier.tmdb.access_token` | `""` | **Your TMDB API access token** — get one free at https://www.themoviedb.org/settings/api. Without it, no metadata lookups happen. |
| `classifier.tmdb.rate_limit` | `20` | Max TMDB requests per second. Keep at or below TMDB's free tier limits. |

### `classifier.llm` — LLM-based classification (advanced)

Uses an OpenAI-compatible API to classify torrents that the rules engine can't figure out. **Only active when both `enabled: true` and an API key are set.**

| Key | Default | What it does |
|---|---|---|
| `classifier.llm.enabled` | `false` | Turn LLM classification on. |
| `classifier.llm.endpoint` | `""` | OpenAI-compatible API base URL, e.g. `https://api.openai.com/v1` or a local server like `http://localhost:11434/v1`. |
| `classifier.llm.api_key` | `""` | Your API key. Required to activate the feature. |
| `classifier.llm.model` | `""` | Model name, e.g. `gpt-4o-mini`. Empty = provider default. |
| `classifier.llm.timeout` | `0` | Request timeout in seconds. `0` = provider default. |
| `classifier.llm.max_retries` | `0` | Retries on failure. `0` = provider default. |
| `classifier.llm.temperature` | `0` | Sampling temperature. `0` = provider default. |
| `classifier.llm.reasoning_effort` | `""` | `low`, `medium` or `high` (for reasoning models). Empty = provider default. |
| `classifier.llm.max_files` | `30` | Max number of torrent files sent to the LLM for analysis. |
| `classifier.llm.prompt` | `""` | Custom system prompt for the LLM classifier. Empty or blank = built-in default prompt (API responses return the built-in default so the UI can show and edit it). Keep the JSON fields (`type`, `base_title`, `date`, `languages`) unchanged. |

### `storage.postgres` — main database

| Key | Default | What it does |
|---|---|---|
| `storage.postgres.host` | `localhost` | Database host. |
| `storage.postgres.port` | `5432` | Database port. |
| `storage.postgres.username` | `postgres` | Database user. |
| `storage.postgres.password` | `""` | Database password. |
| `storage.postgres.database` | `hexmagnet` | Database name. |
| `storage.postgres.ssl_mode` | `disable` | SSL behavior: `disable`, `allow`, `prefer`, `require`, `verify-ca`, `verify-full`. Use `require` or higher when the database is remote or sensitive. |
| `storage.postgres.connection_timeout` | `0` | Connect timeout in seconds. `0` = no timeout. |
| `storage.postgres.ssl_cert_path` | `""` | Client certificate file (for `verify-ca` / `verify-full`). |
| `storage.postgres.ssl_key_path` | `""` | Client key file. |
| `storage.postgres.ssl_root_cert_path` | `""` | CA certificate file. |
| `storage.postgres.max_connections` | `25` | Max simultaneous database connections. Raise for heavy usage. |

### `storage.queue` — how processing jobs are delivered

| Key | Default | What it does |
|---|---|---|
| `storage.queue.backend` | `memory` | `memory` = jobs are processed inside the app (simplest, no extra service). `kafka` = use a Kafka cluster for async processing. |
| `storage.queue.kafka.brokers` | `["localhost:9092"]` | Kafka broker addresses (list of `host:port`), only used with `kafka` backend. |

Switching backends re-routes processing automatically. If you switch to `kafka`, Kafka must be running and reachable, or saves will be rejected.

### `storage.search` — where search happens

| Key | Default | What it does |
|---|---|---|
| `storage.search.backend` | `postgresql` | `postgresql` = search through the main database (simplest). `elasticsearch` = use Elasticsearch for faster, richer search. |
| `storage.search.elasticsearch.addresses` | `["http://localhost:9200"]` | Elasticsearch node addresses, used only with the `elasticsearch` backend. |
| `storage.search.elasticsearch.embedding.endpoint` | `http://localhost:11434/v1` | OpenAI-compatible embeddings API (e.g. local Ollama). Used for semantic search. |
| `storage.search.elasticsearch.embedding.api_key` | `""` | API key for the embeddings endpoint, if it requires one. |
| `storage.search.elasticsearch.embedding.model` | `bge-m3` | Embedding model name. Must be available on the endpoint. |
| `storage.search.elasticsearch.embedding.dimensions` | `1024` | Vector dimensions — **must match the model's output size**. |
| `storage.search.elasticsearch.embedding.instruction_enabled` | `false` | Enable instruction-based embeddings (needed for models like `bge-m3` to work correctly in some setups). |

Switching the search backend re-indexes all torrents automatically.

### `torznab` — Servarr / Prowlarr integration

| Key | Default | What it does |
|---|---|---|
| `torznab.enabled` | `false` | Serve the Torznab API so Lidarr / Radarr / Sonarr / Readarr and Prowlarr can use HexMagnet as an indexer. |
| `torznab.api_key` | `""` | If set, every request must pass `?apikey=`. Leave empty for no key. |
| `torznab.path` | `/torznab` | Base path; the API is served at `{path}/api`, e.g. `http://host:3333/torznab/api`. Applies immediately after save (no restart). |
| `torznab.max_results` | `100` | Maximum results returned per search request. |
| `torznab.categories` | `["*"]` | Newznab category IDs exposed to clients (2000 movies, 3000 music, 5000 TV, 7000 books, 8000 other, …). `*` = all. The list is also enforced on searches: requests for restricted categories return an empty result set. |
| `torznab.trust_proxy_headers` | `true` | Honor `X-Forwarded-Proto` / `X-Forwarded-Host` when building enclosure and download links. Disable when the server is directly exposed, so clients cannot spoof the base URL. |

Search results include magnet links, seeders/peers, and a download URL for the
`.torrent` file (with any configured `server.embed_trackers` embedded), so
Servarr apps can grab and auto-download content found by the crawler.

### `webhooks` — pushing events to external services

| Key | Default | What it does |
|---|---|---|
| `webhooks.enabled` | `false` | Post classified torrent events to the configured URLs. |
| `webhooks.urls` | `[]` | Endpoints that receive every event as a JSON `POST`, one per line. |
| `webhooks.events` | `["classified"]` | Which event types to deliver. Currently only `classified` (a torrent was classified and stored). |
| `webhooks.categories` | `[]` | Filter by classified content type: only torrents whose `content_type` (`movie`, `tv_show`, `music`, `ebook`, `comic`, `audiobook`, `game`, `software`, `other`, `unknown`, `adult`) is in the list are delivered. Empty = all types. A torrent without a content type counts as `unknown`. |
| `webhooks.title_patterns` | `[]` | Filter by torrent name: one RE2 pattern per line, case-insensitive; the event is delivered when at least one pattern matches the name. Empty = no title filter. |
| `webhooks.filename_patterns` | `[]` | Filter by file paths: one RE2 pattern per line, case-insensitive, matched against every file path (parts joined with `/`); the event is delivered when at least one pattern matches at least one file. Empty = no filename filter. |
| `webhooks.timeout` | `10s` | Per-request timeout. |
| `webhooks.max_retries` | `3` | Retries with exponential backoff before an event is dropped (first attempt is not a retry). |
| `webhooks.base_url` | `""` | Public base URL of this server. When set, the payload includes a `torrent_url` link to download the `.torrent` file. |
| `webhooks.headers` | `{}` | Extra HTTP headers sent with every delivery, e.g. `Authorization: Bearer <token>`. Values are masked in the UI; re-saving a masked value keeps the stored one. |
| `webhooks.queue_size` | `1000` | Async delivery queue capacity; when full, new events are dropped (and logged) instead of blocking classification. Applies immediately after save (pending events are migrated). |

Each event payload looks like:

```json
{
  "event": "classified",
  "info_hash": "abcdef...",
  "name": "Artist - Album (2022) FLAC",
  "size": 123456789,
  "content_type": "music",
  "content_source": null,
  "content_id": null,
  "languages": ["en"],
  "seeders": 12,
  "leechers": 3,
  "files_count": 8,
  "files": ["Artist - Album (2022) FLAC/01 - Track.flac", "Artist - Album (2022) FLAC/cover.jpg"],
  "magnet": "magnet:?xt=urn:btih:...",
  "torrent_url": "http://localhost:3333/api/torrents/abcdef.../download",
  "created_at": "2026-09-03T12:00:00Z"
}
```

The `categories`, `title_patterns` and `filename_patterns` filters are
combined with AND; patterns within a list are OR-ed. They only affect which
events are delivered — the payload is identical otherwise. Filters apply to
events enqueued after a config save; already-queued events are unaffected.

See [Usage — Webhook integration](usage.md#webhook-integration) for consumer examples.

---

## Example files

**Minimal config** (everything else falls back to defaults):

```yaml
server:
  ip: ""
  port: 3333

dht:
  port: 3334

storage:
  postgres:
    host: localhost
    username: postgres
    password: mypassword
    database: hexmagnet
```

**A fuller example**:

```yaml
server:
  port: 3333
  embed_trackers:
    - udp://tracker.opentrackr.org:1337/announce
  log:
    file_output_level: info
    file_rotator:
      path: ./logs
      max_backups: 10
      format: json

dht:
  reseed_bootstrap_nodes_interval: 60s
  requester:
    hash_discover_limit: 20

classifier:
  tmdb:
    enabled: true
    access_token: "your-tmdb-token"
  llm:
    enabled: true
    endpoint: https://api.openai.com/v1
    api_key: "your-api-key"
    model: gpt-4o-mini

storage:
  postgres:
    host: postgres.example.com
    username: hexmagnet
    password: "secret"
    ssl_mode: require
  search:
    backend: elasticsearch
    elasticsearch:
      addresses:
        - http://localhost:9200
```

See `example/hexmagnet.yaml` in the repository for a fully commented template with every option.

## Applying changes

| Change | Applies |
|---|---|
| Everything on the Config page | Immediately after save (no restart) |
| `server.ip` / `server.port` | Only after restart |
| Config file edits | After restart (or via the dashboard) |
| Environment variables | After restart (or re-run the process) |
