# Troubleshooting

Step-by-step fixes for common problems. If your issue isn't here, the logs are the best place to start — see [Reading the logs](#reading-the-logs) at the end.

---

## The service won't start

### "config directory ... does not exist"

The parent folder of your config file doesn't exist.

1. Note the path in the error message
2. The folder must exist **before** startup — HexMagnet won't create it
3. Create it, e.g. `mkdir -p /srv/hexmagnet/app`, and start again
4. If you set `HEXMAGNET_CONFIG_FILE` or `-c`, double-check that path too

### "validate ... failed" or "config error" at startup

One of the settings in `hexmagnet.yaml` (or an env var) is invalid. The message names the key — for example:

```
config error: validate "server": failed on the 'lte' tag ... port
```

1. Read the key from the error, e.g. `server.port`
2. Open your config file and check the value against the [settings reference](configuration.md#settings-reference) (range, allowed values, spelling)
3. Common culprits:
   - `dht.reseed_bootstrap_nodes_interval` written as a plain number — use duration format: `60s`
   - `server.log.console_level` misspelled — must be `debug`, `info`, `warn` or `error`
   - `storage.postgres.ssl_mode` not in the allowed list
   - An empty `server.torrent_file_path` — it is required
4. Fix the value and start again

### "error coercing env key ... to type"

An environment variable has the wrong format. The message shows the variable — for example:

```
error coercing env key 'DHT_RESEED_BOOTSTRAP_NODES_INTERVAL' with value '60' to type time.Duration
```

- Durations need Go duration format: `60s`, `5m`, `1h`
- Booleans: `true` / `false` (or `1` / `0`)
- Lists are comma-separated

### "address already in use"

Another program is using port 3333 (or 3334). Either stop that program, or change `server.port` / `dht.port` in your config.

---

## The dashboard rejects my config

When you click Save on the Config page, HexMagnet tests your new values first. If a test fails, **nothing is saved** and an error explains why. Here's how to fix each kind of failure:

### "torrent_file_path is not writable" / log path not writable

HexMagnet creates a test file in the folder to check.

1. Check the path exists: `ls -ld /path/to/folder`
2. The service process must have write permission (check ownership if running under Docker)
3. With Docker Compose, the path must live inside a mounted volume (see [Installation](installation.md))
4. Fix the permission and save again

### "postgres" connection failed

The PostgreSQL settings are wrong or the server is unreachable.

1. Check the database server is running and reachable from the same network (host + port)
2. Verify username, password and database name — with Docker, these come from the compose `environment` section
3. If the database is remote, set `storage.postgres.ssl_mode` to `require` (or higher) and provide the cert files for `verify-ca` / `verify-full`
4. Try connecting with your own client first (e.g. `psql`), then save again

### "elasticsearch" unreachable

Only checked when the search backend is Elasticsearch.

1. Check `storage.search.elasticsearch.addresses` is right
2. Verify Elasticsearch is running and healthy
3. If it's behind a proxy or firewall, make sure the address is reachable from the HexMagnet container

### "kafka" broker unreachable

Only checked when the queue backend is Kafka.

1. Check `storage.queue.kafka.brokers` — each entry must be `host:port`
2. Verify Kafka is healthy (`docker compose ps` shows it healthy)
3. Kafka advertising must match the address HexMagnet uses — with Docker Compose the advertised listener is `kafka:9092`, so `storage.queue.kafka.brokers` must say `kafka:9092`, not `localhost:9092`
4. The error reports **every** unreachable broker — fix them all

### "tmdb" token invalid

Only checked when you change TMDB settings (and skipped if TMDB is disabled or the token is empty).

1. Get a valid token at https://www.themoviedb.org/settings/api
2. Paste it into `classifier.tmdb.access_token` and save again
3. Tokens are checked live — if it was revoked or expired, get a new one

### "llm" endpoint unreachable

Only checked when the LLM classifier is enabled and an endpoint is set.

1. Verify the URL in `classifier.llm.endpoint` — it must be an OpenAI-compatible base URL
2. From the machine running HexMagnet, test it: `curl -v <endpoint>`
3. If the API is behind a key-only route, `classifier.llm.api_key` must be set for the check to pass
4. Save again once the endpoint answers

---

## I see no torrents in search

### The service is healthy but nothing appears

1. Open the dashboard and look at the **crawler overview** — is it discovering hashes?
2. If discovery is zero:
   - Check that port `3334/udp` is reachable from the internet (firewall / NAT). Many home networks need port forwarding
   - Verify `dht.bootstrap_nodes` are reachable from your network
3. If discovery is slow:
   - Raise `dht.requester.hash_discover_limit` (more hashes per cycle)
   - Make sure `dht.requester.request_limit` is not `0` (0 means unlimited — that's fine) and not overly low
4. Wait — it can take a while before the first batch is fully classified and searchable. Watch the queue panel for progress

### Torrents existed but a search returns nothing

1. Did you switch the search backend recently? Switching re-indexes everything — large libraries take time. The dashboard save would have told you if the switch failed
2. Search with fewer filters first (empty query, no filters, sort by newest)
3. If you used a pattern filter (`torrent_filter`) set to `discard`, matching torrents are intentionally removed — check the mode

---

## Classification / metadata problems

### Torrents have no movie/TV metadata

1. Check `classifier.tmdb.enabled` is `true`
2. Check `classifier.tmdb.access_token` is set and valid (a token is **required**)
3. Check the TMDB rate limit isn't too low for your crawler speed
4. Existing torrents are not re-enriched automatically — open one and hit **Reprocess**

### The LLM classifier never runs

1. Both `classifier.llm.enabled` **and** `classifier.llm.api_key` must be set — one alone is not enough
2. Verify the endpoint is reachable and the model name is valid for that provider
3. If requests fail silently: raise `classifier.llm.timeout` and `classifier.llm.max_retries`
4. Test the same request with `curl` against the endpoint — this isolates provider problems
5. Reprocess an existing torrent to see it in action

### Semantic search doesn't work (Elasticsearch backend)

Semantic search needs the embedding endpoint (usually local Ollama).

1. Check Ollama (or your embeddings server) is running: `curl http://localhost:11434/v1` — if you run HexMagnet in Docker, this is `http://host.docker.internal:11434/v1` on macOS/Windows, or the host IP on Linux
2. Pull the model: `ollama pull bge-m3` (or whatever `storage.search.elasticsearch.embedding.model` says)
3. **`dimensions` must match the model's output size** (bge-m3 = 1024). Mismatches cause indexing errors
4. For `bge-m3` in particular, set `instruction_enabled: true` if queries return poor results

---

## After switching backends

### Switched search to Elasticsearch but search is broken

1. The switch only saves if Elasticsearch is reachable — if the dashboard rejected it, see the Elasticsearch section above
2. A full re-index runs in the background after the switch; large libraries take time
3. Check the dashboard queue panel for indexing activity

### Switched queue to Kafka but nothing processes

1. Check `storage.queue.kafka.brokers` uses the addresses Kafka actually advertises (with Docker Compose: `kafka:9092`)
2. Verify Kafka is healthy: `docker compose ps kafka`
3. Jobs should re-route automatically; if the queue panel stays empty, restart the service once

---

## Web UI problems

### The web UI shows a blank page or an error

1. If you run the official Docker image, this shouldn't happen — try a hard refresh
2. If you built from source with `go build`, the UI was not included. Use `make build` (it builds the web UI first). The server logs a clear message about the missing `dist` folder
3. Check the browser console / network tab for failed requests to `/graphql`

---

## Reading the logs

Logs are your best friend for anything not covered here.

**Console** — set `server.log.console_level` to `debug` for maximum detail (temporarily, then set it back).

**Log files** — enable with `server.log.file_output_level: info`, stored in `server.log.file_rotator.path` (`./logs` by default). Format `json` is easiest to search with tools.

**Docker** — see logs with:

```bash
docker compose -f deployment/docker-compose.yml logs -f hexmagnet
```

**Where to look for specific problems:**

| Problem | Search the logs for |
|---|---|
| Crawler not working | `dht`, `bootstrap`, `crawler` |
| Classification | `classifier`, `tmdb`, `llm` |
| Search / indexing | `indexer`, `elasticsearch`, `reindex` |
| Database | `postgres`, `pool`, `migration` |
| Queue | `queue`, `kafka`, `consumer` |
| Startup | `starting`, `listening`, `fatal` |

If you still can't find the cause, include the relevant log lines (with `console_level: debug`) when asking for help.
