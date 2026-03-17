# snowflake-uuid

A Twitter-style [Snowflake](https://blog.twitter.com/engineering/en_us/a/2010/announcing-snowflake) ID generator for Go. Produces globally unique, time-sortable 64-bit identifiers with zero external dependencies.

Use it as an **importable Go library**, a **Kubernetes sidecar**, or a **systemd service**.

## Bit Layout

```
 63   62                   22 21        12 11          0
 ┌─┬──────────────────────────┬───────────┬─────────────┐
 │0│     timestamp (41 bit)   │ node (10) │  seq (12)   │
 └─┴──────────────────────────┴───────────┴─────────────┘
  │            │                    │            │
  │            │                    │            └─ 4,096 IDs per ms per node
  │            │                    └────────────── 1,024 unique nodes
  │            └─────────────────────────────────── ms since custom epoch (~69.7 years)
  └──────────────────────────────────────────────── always 0 (positive int64)
```

**Default epoch:** `2024-01-01 00:00:00 UTC` — IDs remain valid until ~2093.

## Features

- **Zero dependencies** — stdlib only, no transitive bloat
- **Goroutine-safe** — mutex-guarded, race-detector clean
- **Zero allocations** — entire hot path is stack-allocated
- **~4.9M IDs/sec** single-goroutine throughput
- **Clock drift tolerance** — spin-waits up to 5ms on backward clock
- **JSON-safe** — IDs marshal as strings to avoid JS precision loss (`Number.MAX_SAFE_INTEGER` is 2^53)

## Quick Start

### As a Go Library

```bash
go get github.com/hra4h03/snowflake-uuid
```

```go
package main

import (
    "fmt"
    "github.com/hra4h03/snowflake-uuid/snowflake"
)

func main() {
    gen, err := snowflake.New(1) // node ID: 0–1023
    if err != nil {
        panic(err)
    }

    id, err := gen.Generate()
    if err != nil {
        panic(err)
    }

    fmt.Println(id)              // "292286451008147456"
    fmt.Println(id.Timestamp())  // ms since epoch
    fmt.Println(id.NodeID())     // 1
    fmt.Println(id.Sequence())   // 0
}
```

#### Custom Epoch

```go
epoch := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
gen, _ := snowflake.New(1, snowflake.WithEpoch(epoch))
```

#### Decompose an ID

```go
id := snowflake.ID(292286451008147456)

id.Timestamp() // 69686521038 (ms since epoch)
id.NodeID()    // 1
id.Sequence()  // 0
id.Time(snowflake.DefaultEpoch) // 2026-03-17T13:22:01.038Z
```

### As an HTTP Service

```bash
# Build and run
go build -o snowflake-server ./cmd/snowflake-server
./snowflake-server -node-id 1

# Or run directly
go run ./cmd/snowflake-server -node-id 1
```

### With Docker

```bash
make docker-build
docker run -e SNOWFLAKE_NODE_ID=1 -p 8080:8080 snowflake-uuid:latest
```

### With systemd

```bash
# Install binary and unit file
make install-systemd

# Configure node ID
echo "SNOWFLAKE_NODE_ID=1" | sudo tee /etc/snowflake-uuid/snowflake-uuid.env

# Start
sudo systemctl start snowflake-uuid
sudo systemctl status snowflake-uuid
```

### As a Kubernetes Sidecar

See [`deploy/k8s/deployment.yaml`](deploy/k8s/deployment.yaml) for a full example. Your application accesses the sidecar at `http://localhost:8080/id`.

## HTTP API

### `GET /id`

Generate a single ID.

```bash
$ curl localhost:8080/id
```
```json
{"id": "292286451008147456"}
```

### `GET /id/batch?count=N`

Generate up to 1000 IDs in a single request.

```bash
$ curl 'localhost:8080/id/batch?count=3'
```
```json
{"ids": ["292286452119638016", "292286452119638017", "292286452119638018"]}
```

### `GET /id/parse?id=<id>`

Decompose an ID into its constituent parts.

```bash
$ curl 'localhost:8080/id/parse?id=292286453935771648'
```
```json
{
  "id": "292286453935771648",
  "timestamp_ms": 69686521038,
  "node_id": 1,
  "sequence": 0,
  "generated_at": "2026-03-17T13:22:01.038Z"
}
```

### `GET /health`

Health check for liveness/readiness probes.

```bash
$ curl localhost:8080/health
```
```json
{"status": "ok", "node_id": 1, "uptime_seconds": 3600}
```

## Configuration

Settings are loaded from flags and environment variables. Flags take precedence.

| Setting | Flag | Env Var | Default | Description |
|---------|------|---------|---------|-------------|
| Node ID | `-node-id` | `SNOWFLAKE_NODE_ID` | — (required) | Unique node identifier (0–1023) |
| Listen address | `-listen-addr` | `SNOWFLAKE_LISTEN_ADDR` | `:8080` | HTTP bind address |
| Auto node ID | `-auto-node-id` | `SNOWFLAKE_AUTO_NODE_ID` | `false` | Derive node ID from environment |
| Epoch | `-epoch` | `SNOWFLAKE_EPOCH_MS` | `1704067200000` | Custom epoch (Unix ms) |

**Auto node ID resolution** (when `-auto-node-id` is enabled):
1. K8s: FNV hash of `HOSTNAME` (pod name) mod 1024
2. Fallback: lower 10 bits of the host's private IP address

## Test Coverage

```
Package                        Coverage
─────────────────────────────────────────
snowflake/                     96.2%
  New                          100.0%
  Generate                     95.0%
  ID.Timestamp                 100.0%
  ID.NodeID                    100.0%
  ID.Sequence                  100.0%
  ID.Time                      100.0%
  ID.MarshalJSON               100.0%
  ID.UnmarshalJSON             91.7%
internal/server/               65.5%
  RegisterRoutes               100.0%
  handleGenerateID             60.0%
  handleBatchIDs               86.7%
  handleParseID                100.0%
  handleHealth                 100.0%
  writeJSON                    100.0%
```

### Test Suite

| Test | What it validates |
|------|-------------------|
| `TestBitLayout` | Correct field placement in the 64-bit ID |
| `TestSignBitAlwaysZero` | 10K IDs — all positive int64 |
| `TestUniqueness` | 1M sequential IDs — zero duplicates |
| `TestMonotonicity` | 100K IDs — each strictly greater than previous |
| `TestConcurrency` | 100 goroutines x 10K IDs — zero duplicates (with `-race`) |
| `TestSequenceOverflow` | Sequence wraps to 0 at next millisecond |
| `TestClockBackwardSmall` | Tolerates ≤5ms backward drift |
| `TestClockBackwardLarge` | Returns error on >5ms backward drift |
| `TestIDDecomposition` | Field extraction round-trip |
| `TestIDMarshalRoundTrip` | JSON string serialization fidelity |
| Handler tests | All HTTP endpoints, error cases, edge cases |

Run the full suite:

```bash
make test          # all tests with race detector
make test-cover    # tests + HTML coverage report
make bench         # throughput benchmarks
```

## Benchmarks

```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro

BenchmarkGenerate-14            4,925,469    244.0 ns/op    0 B/op    0 allocs/op
BenchmarkGenerateParallel-14    4,924,578    244.0 ns/op    0 B/op    0 allocs/op
```

~**4.9 million IDs/sec** per node, with zero heap allocations.

Theoretical maximum: 4,096 IDs/ms = 4,096,000 IDs/sec (limited by 12-bit sequence space).

## Make Targets

```bash
make build           # Build server binary to ./bin/
make test            # Run all tests with race detector
make test-cover      # Tests + HTML coverage report
make bench           # Run benchmarks
make lint            # Run go vet
make docker-build    # Build Docker image
make install-systemd # Install binary + systemd unit
make clean           # Remove build artifacts
```

## License

MIT
