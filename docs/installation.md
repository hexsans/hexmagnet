# Installation

HexMagnet is a single binary that starts everything on its own — HTTP server, DHT crawler and processors. You can run it with Docker Compose, plain Docker, or directly from source.

## Requirements

- **Docker Compose** — for the compose setups below
- **Go 1.26+** — only needed for building from source
- **Ports**: make sure these are free and reachable
  - `3333` (TCP) — web UI and API
  - `3334` (TCP + UDP) — BitTorrent network (the DHT node listens on UDP; TCP is used for metadata downloads)
- **Memory**: 1 GB is the practical minimum for the app itself; Elasticsearch and Kafka (if used) need more — the production compose file works well on a machine with 4 GB+.

## Docker Compose (recommended)

### Local / development stack

This builds the image from your current checkout (with debug tooling), and starts HexMagnet plus PostgreSQL, Elasticsearch and Kafka. Data is stored under `.testing/` in the project.

```bash
docker compose -f deployment/local/docker-compose.yml up -d
```

### Production stack

This pulls the prebuilt image from GHCR and starts HexMagnet with PostgreSQL, Elasticsearch and Kafka. It assumes your config and data live on the host under `/srv/hexmagnet/`:

```bash
docker compose -f deployment/docker-compose.yml up -d
```

A few things to know about the production compose file:

- Your config lives at `/srv/hexmagnet/app/hexmagnet.yaml` (mounted into `/app`)
- The container data dirs (`/app/data`, `/app/logs`) are inside the mounted `/srv/hexmagnet/app` volume, so torrent files and logs survive restarts
- Kafka, Elasticsearch and PostgreSQL each get their own volume under `/srv/hexmagnet/`
- Default database credentials in the compose file are `postgres` / `postgres` — change them if the machine is not fully isolated

## Docker (no Compose)

```bash
docker pull ghcr.io/hexsans/hexmagnet:latest
docker run -p 3333:3333 -p 3334:3334/tcp -p 3334:3334/udp \
  ghcr.io/hexsans/hexmagnet:latest
```

This runs with compiled-in defaults — PostgreSQL is expected at `localhost:5432`. If you have PostgreSQL elsewhere, mount a config file and point the app at it:

```bash
docker run -p 3333:3333 -p 3334:3334/tcp -p 3334:3334/udp \
  -v /srv/hexmagnet/app:/app \
  -e HEXMAGNET_CONFIG_FILE=/app/hexmagnet.yaml \
  ghcr.io/hexsans/hexmagnet:latest
```

The first time it starts it will create a complete default config at `/app/hexmagnet.yaml`. Edit that file and restart to apply your settings.

## Build from source

Requires Go 1.26+.

```bash
git clone https://github.com/hexsans/hexmagnet.git
cd hexmagnet
make build
./hexmagnet
```

`make build` compiles both the web UI and the Go binary. To point the binary at a config file:

```bash
./hexmagnet -c /path/to/config.yaml
```

## First start

On the very first start, HexMagnet:

1. Looks for its config file (see [Configuration](configuration.md#where-the-config-file-lives))
2. **Creates it automatically with all defaults** if it doesn't exist
3. Applies database migrations (PostgreSQL) if needed
4. Starts the HTTP server, DHT crawler and processors

You should see `http server listening` in the logs when everything is ready — then open the web UI at `http://localhost:3333/`.

The Torznab API (for Servarr apps and Prowlarr) is served on the same HTTP port — no extra ports to open. See [Usage — Servarr integration](usage.md#servarr-integration-lidarr--radarr--sonarr--readarr).

## Where your data lives

| What | Default location |
|---|---|
| Config file | `./hexmagnet.yaml` (next to the binary / in the container's `/app`) |
| Downloaded `.torrent` files | `./data/torrents/` (configurable via `server.torrent_file_path`) |
| Log files | `./logs/` (configurable via `server.log.file_rotator.path`) |
| PostgreSQL data | Docker volume (compose) or your own server |
| Elasticsearch data | Docker volume (compose) or your own server |
| Kafka data | Docker volume (compose) or your own server |

## Upgrading

1. Pull the new image: `docker compose -f deployment/docker-compose.yml pull hexmagnet`
2. Restart: `docker compose -f deployment/docker-compose.yml up -d`
3. Database migrations run automatically on startup — no manual steps
4. Your `hexmagnet.yaml` and data volumes are untouched by upgrades. If a setting has changed between versions, the dashboard may show a validation error on save — see [Troubleshooting](troubleshooting.md)

## Uninstalling

Stop the stack and remove volumes only if you want to delete all data:

```bash
docker compose -f deployment/docker-compose.yml down -v
```

The `-v` flag deletes the PostgreSQL, Elasticsearch, Kafka and app volumes — your indexed torrents and metadata are gone for good.
