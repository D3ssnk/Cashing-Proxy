# Caching Proxy Server

A CLI tool that runs a caching proxy server. Forwards HTTP requests to an origin server, caches responses on disk, and returns cached responses on subsequent identical requests — no network round trip required.

## How It Works

The proxy sits between the client and the origin server:

```
First request (MISS):
Client → Proxy → Origin Server
                     ↓
                   Cache

Subsequent requests (HIT):
Client → Proxy → Cache
```

On a **MISS**, the proxy forwards the request to the origin, stores the response body, headers, and status code in a local `cache.json` file, then returns the response to the client.

On a **HIT**, the proxy reads the stored response from disk and returns it directly — significantly faster than hitting the origin.

The cache persists across server restarts. Responses are stored in `cache.json` in the working directory.

## Usage

### Start the server

```bash
go run main.go --port <number> --origin <url>
```

```bash
go run main.go --port 3000 --origin https://dummyjson.com
```

The server will start on the specified port and forward all incoming requests to the origin.

### Making requests

```bash
curl -v http://localhost:3000/products
```

The response will include an `X-Cache` header indicating whether it was served from cache or the origin:

```
X-Cache: MISS   # fetched from origin, now cached
X-Cache: HIT    # served from cache
```

### Clear the cache

```bash
go run main.go --clear-cache
```

Deletes `cache.json`. The server does not need to be running.

## Flags

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--port` | int | 3000 | Port to run the proxy server on |
| `--origin` | string | — | URL of the origin server to forward requests to |
| `--clear-cache` | bool | false | Clears the cache file and exits |

## Implementation Details

**Cache key** — requests are keyed by full URI including query parameters (e.g. `/products?limit=10`), so requests with different query strings are cached independently.

**Stored per entry** — response body, status code, and all origin headers are cached and replayed on HITs, so the client receives an identical response either way.

**Concurrency** — incoming requests are handled concurrently by Go's HTTP server. The cache map is protected by a `sync.RWMutex`: multiple goroutines can read simultaneously, but writes are exclusive. This prevents data races under concurrent load.

**Persistence** — the cache is serialised as JSON and written to disk on every new entry. On startup the cache is loaded from disk, so responses cached in previous sessions are available immediately.

## Requirements

Go 1.21+

## Running Tests

```bash
go test ./...
```yeah