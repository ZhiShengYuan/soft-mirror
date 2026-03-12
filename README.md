# file-host

A lightweight HTTP server for hosting and distributing executable binaries across platforms and versions. Built with Go and [Gin](https://github.com/gin-gonic/gin).

## Features

- **Multi-platform binary hosting** -- store binaries for linux, darwin, and windows on amd64 and arm64
- **Semantic versioning** -- version resolution supports exact versions, constraints (`^1.0`, `~1.2`, `>=1.0 <2.0`), and `latest`
- **Auto-detection** -- detects client OS and architecture from User-Agent for automatic downloads
- **HMAC authentication** -- upload and delete operations are protected by HMAC-SHA256 request signing with replay protection
- **Atomic uploads** -- binaries are written to a temp file and renamed into place
- **ETag and Range support** -- direct downloads support conditional requests and partial content via `http.ServeContent`
- **Web UI** -- browse hosted programs and versions through HTML pages at `/` and `/programs/:name`
- **Security headers** -- responses include `X-Content-Type-Options`, `X-Frame-Options`, `Content-Security-Policy`, and `X-XSS-Protection`
- **Graceful shutdown** -- handles SIGINT/SIGTERM with a 30-second drain period

## Prerequisites

- **Go 1.25.0** or later

## Build

```bash
go build -o bin/server ./cmd/server
```

## Run

The server requires the `HMAC_SECRET` environment variable (or config file equivalent) to start.

```bash
# Minimal
HMAC_SECRET=your-secret ./bin/server

# With go run
HMAC_SECRET=your-secret go run ./cmd/server

# With all options
HMAC_SECRET=your-secret \
  LISTEN_ADDR=:9090 \
  DATA_DIR=/var/lib/file-host \
  LOG_LEVEL=debug \
  ./bin/server
```

## Configuration

Settings are loaded in order of precedence: **environment variables** override **config file** values, which override **defaults**.

Point the server at a YAML config file with `CONFIG_FILE`:

```bash
CONFIG_FILE=/etc/file-host/config.yaml HMAC_SECRET=your-secret ./bin/server
```

### Config reference

| Field | Env var | Default | Description |
|---|---|---|---|
| `listen_addr` | `LISTEN_ADDR` | `:8080` | Address and port to listen on |
| `data_dir` | `DATA_DIR` | `./data` | Root directory for binary storage |
| `hmac_secret` | `HMAC_SECRET` | _(required)_ | Shared secret for HMAC-SHA256 request signing |
| `hmac_max_drift` | `HMAC_MAX_DRIFT` | `5m` | Maximum allowed clock drift for HMAC timestamps (must be between 0 and 15m) |
| `max_upload_size` | `MAX_UPLOAD_SIZE` | `536870912` (512 MiB) | Maximum upload body size in bytes |
| `log_level` | `LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `log_format` | `LOG_FORMAT` | `json` | Log output format: `json`, `text` |
| `log_file` | `LOG_FILE` | _(empty, stdout)_ | Path to log file; empty means stdout |
| `trusted_proxies` | `TRUSTED_PROXIES` | _(empty)_ | Comma-separated list of trusted proxy IPs/CIDRs |

### Example config.yaml

```yaml
listen_addr: ":8080"
data_dir: "/var/lib/file-host/data"
hmac_secret: "change-me"
hmac_max_drift: 5m
max_upload_size: 536870912
log_level: info
log_format: json
trusted_proxies:
  - "10.0.0.0/8"
```

## Storage layout

Binaries are stored on the filesystem under `data_dir` with the following directory structure:

```
data/
  {program}/
    {version}/
      {os}/
        {arch}/
          {program}          # linux, darwin
          {program}.exe      # windows
```

For example, uploading `myapp` version `1.2.3` for `linux/amd64` creates:

```
data/myapp/1.2.3/linux/amd64/myapp
```

Deleting a binary cleans up empty parent directories (arch, os, version) automatically. Deleting an entire version removes the version directory tree.

## API documentation

See [docs/api.md](docs/api.md) for the full API reference, including all endpoints, HMAC signing protocol, and curl examples.
