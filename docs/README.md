# HexMagnet User Guide

HexMagnet is a self-hosted BitTorrent indexer, DHT crawler, content classifier and torrent search engine. It discovers torrents from the BitTorrent DHT network, classifies them into movies, TV shows, music and more, enriches them with metadata, and gives you a fast search UI — all on your own hardware.

This guide covers everything you need to run, configure and fix HexMagnet. No technical knowledge required.

## Documentation

| Doc | What it covers |
|---|---|
| [Installation](installation.md) | Docker Compose, Docker, and binary installs; ports; where your data lives; upgrading |
| [Usage](usage.md) | The web UI step by step: dashboard, search, torrent details, downloading, reprocessing — plus Torznab / Servarr integration (Lidarr, Radarr, Sonarr, Readarr, Prowlarr) |
| [Configuration](configuration.md) | Every setting explained in plain language — what it does, valid values, and how to change it |
| [Troubleshooting](troubleshooting.md) | Step-by-step fixes for the most common problems, including wrong config and save errors |

## Quick start

If you just want to try it:

```bash
docker compose -f deployment/local/docker-compose.yml up -d
```

Open **http://localhost:3333/** and start searching. See [Installation](installation.md) for the full picture.
