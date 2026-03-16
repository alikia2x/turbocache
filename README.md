# Turbocache

Turborepo Remote Cache Server implemented in Go with Gin.

## Features

-   Fully compatible with Turborepo Remote Cache API
-   Token-based authentication
-   Local filesystem storage
-   Docker support

## Quick Start

```bash
# Run locally
make run

# Or with custom config
TURBO_TOKEN=your-token CACHE_DIRECTORY=/data/cache PORT=3000 ./bin/turbocache
```

## Configuration

| Environment Variable    | Default   | Description                       |
| ----------------------- | --------- | --------------------------------- |
| `TOKEN` / `TURBO_TOKEN` | -         | Authentication token              |
| `CACHE_DIRECTORY`       | `./cache` | Cache storage directory           |
| `PORT`                  | `3000`    | Server port                       |
| `MAX_CACHE_SIZE`        | `0`       | Max cache size in MB (0=disabled) |
| `MAX_CACHE_COUNT`       | `0`       | Max artifact count (0=disabled)   |
| `EVICT_BATCH`           | `10`      | Artifacts to evict per cleanup    |

Automatically loads `.env` file if present (existing env vars take precedence).

## API Endpoints

All endpoints prefixed with `/v8`

| Method | Path                | Description           |
| ------ | ------------------- | --------------------- |
| GET    | `/artifacts/status` | Get caching status    |
| HEAD   | `/artifacts/{hash}` | Check artifact exists |
| GET    | `/artifacts/{hash}` | Download artifact     |
| PUT    | `/artifacts/{hash}` | Upload artifact       |
| POST   | `/artifacts`        | Query artifacts       |
| POST   | `/artifacts/events` | Record cache events   |

## Development

```bash
# Build
make build

# Test
make test

# Lint
make lint

# Coverage report
make coverage

# Clean
make clean
```

## Docker

```bash
# Build image
make docker

# Run container
docker run -p 3000:3000 \
  -e TURBO_TOKEN=your-token \
  -v /path/to/cache:/app/cache \
  turbocache:latest
```

## License

MIT
