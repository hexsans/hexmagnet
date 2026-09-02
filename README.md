# HexMagnet

A self-hosted BitTorrent indexer, DHT crawler, content classifier and torrent search engine with web UI, REST API and Servarr stack integration.

Forked from [bitmagnet](https://github.com/bitmagnet-io/bitmagnet).

## Quick Start

```bash
docker compose -f deployment/local/docker-compose.yml up -d
```

Once running, open the web UI at http://localhost:3333/ and start searching. The dashboard also lets you monitor activity and change settings live on the Config page.

## Installation

### Docker Compose (recommended)

For local development with the full stack, use:

```bash
docker compose -f deployment/local/docker-compose.yml up -d
```

For production deployment, use:

```bash
docker compose -f deployment/docker-compose.yml up -d
```

### Docker

```bash
docker pull ghcr.io/hexsans/hexmagnet:latest
docker run -p 3333:3333 \
  ghcr.io/hexsans/hexmagnet:latest
```

### Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/hexsans/hexmagnet.git
cd hexmagnet
make build
./hexmagnet
```

Optionally specify a config file:

```bash
./hexmagnet -c /path/to/config.yaml
```

## Usage

The binary starts the full service directly — HTTP server, DHT crawler and processors all run automatically. By default the config path resolves to the `HEXMAGNET_CONFIG_FILE` env var or `./hexmagnet.yaml`.

The web UI at `http://localhost:3333/` is the primary interface for searching, monitoring, and changing settings via the Config page. Most settings take effect immediately; only the HTTP listen address (`server.ip` / `server.port`) requires a restart.

## Configuration

Configuration is resolved from multiple sources in increasing priority order:

1. **Default values** (compiled-in)
2. **Runtime config file** — `./hexmagnet.yaml` (or `$HEXMAGNET_CONFIG_FILE`) — persisted from the dashboard
3. **Local config file** — `./config.yml` (optional, CWD)
4. **Environment variables** — `UPPER_SNAKE_CASE` of config keys

See `example/hexmagnet.yaml` for a fully commented reference of every option.

### Environment variables

| Env var | Default | Description |
|---|---|---|
| `HEXMAGNET_CONFIG_FILE` | `./hexmagnet.yaml` | Path to runtime config file |

### Key config options

| Group | Key | Default | Description |
|---|---|---|---|
| **server** | `ip` + `port` | `""` `3333` | Listen address (`""` = all interfaces) |
| **server** | `log.console_level` | `"info"` | Log level (debug, info, warn, error) |
| **server** | `log.file_output_level` | `"off"` | Write logs to file (debug, info, warn, error, off) |
| **server** | `log.file_rotator` | `./logs`, 5, text | Log file path, backups, format (text/json) |
| **server** | `embed_trackers` | `[]` | Tracker announce URLs to embed in .torrent downloads |
| **server** | `torrent_file_path` | `./data/torrents` | Directory where torrent files are stored |
| **dht** | `port` | `3334` | UDP listen port for the DHT node |
| **dht** | `bootstrap_nodes` | 6 default nodes | Initial nodes used to join the DHT network |
| **dht** | `reseed_bootstrap_nodes_interval` | `60` | How often to re-query bootstrap nodes (seconds) |
| **dht** | `requester.*` | `50` QPS | Request limits, re-scrape threshold, hash discovery |
| **dht** | `responder.*` | enabled | DHT query responder settings and rate limits |
| **classifier** | `concurrency` | `10` | Max concurrent classification runs |
| **classifier** | `torrent_filter` | `off` | Pre-classification filter (off/process/discard) |
| **classifier** | `llm.*` | disabled | LLM-based classification (endpoint, API key, model) |
| **tmdb** | `enabled` + `access_token` | `true` `""` | TMDB metadata enrichment (requires a token from themoviedb.org/settings/api) |
| **storage** | `postgres.*` | `localhost:5432/hexmagnet` (user `postgres`) | PostgreSQL connection |
| **storage** | `queue.backend` | `memory` | Message queue backend (memory/kafka) |
| **storage** | `queue.kafka.brokers` | `["localhost:9092"]` | Kafka broker addresses |
| **storage** | `search.backend` | `postgresql` | Search backend (postgresql/elasticsearch) |
| **storage** | `search.elasticsearch.*` | `http://localhost:9200` | Elasticsearch addresses and embedding settings |

## Third-Party Licenses

This project incorporates code derived from the following open-source projects:

- **alexliesenfeld/health** — MIT License (https://github.com/alexliesenfeld/health)
- **anacrolix/dht**, **anacrolix/missinggo**, **anacrolix/torrent** — MPL 2.0 (https://github.com/anacrolix/torrent)
